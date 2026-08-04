package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/stretchr/testify/assert"
)

type stubSnsClient struct {
	input *sns.PublishInput
	err   error
	calls int
}

func (s *stubSnsClient) Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error) {
	s.calls++
	s.input = params
	return &sns.PublishOutput{}, s.err
}

func TestNewChangeNotification(t *testing.T) {
	// Given
	moved := baseGame()
	moved.PlayDate = "2025-09-21 15:00:00"
	changes := []GameChange{{
		Game: moved,
		Changes: []FieldChange{
			{Field: FieldPlayDate, Old: "2025-09-20 17:00:00", New: "2025-09-21 15:00:00"},
			{Field: FieldHall, Old: "Salle du Collège, Fribourg", New: "Gymnase, Bulle"},
		},
	}}
	detectedAt := time.Date(2026, 7, 24, 16, 0, 0, 0, time.UTC)

	// When
	notification := newChangeNotification(changes, detectedAt)

	// Then
	assert.Equal(t, 1, notification.Version)
	assert.Equal(t, "volley.matches.changed", notification.Type)
	assert.Equal(t, detectedAt, notification.DetectedAt)
	if assert.Len(t, notification.Games, 1) {
		game := notification.Games[0]
		assert.Equal(t, moved.GameId, game.GameId)
		assert.Equal(t, "2025-09-21 15:00:00", game.PlayDate)
		assert.Equal(t, "Gibloux Volley F1", game.HomeTeam)
		assert.Equal(t, "Volley Fribourg", game.AwayTeam)
		assert.Equal(t, "Salle du Collège, Fribourg", game.Hall)
		assert.Len(t, game.Changes, 2)
	}

	raw, err := json.Marshal(notification)
	assert.Nil(t, err)

	var decoded map[string]any
	err = json.Unmarshal(raw, &decoded)

	// Then
	assert.Nil(t, err)
	assert.Contains(t, decoded, "version")
	assert.Contains(t, decoded, "type")
	assert.Contains(t, decoded, "detectedAt")
	assert.Contains(t, decoded, "games")
	game := decoded["games"].([]any)[0].(map[string]any)
	assert.Contains(t, game, "gameId")
	assert.Contains(t, game, "changes")
	change := game["changes"].([]any)[0].(map[string]any)
	assert.Equal(t, "playDate", change["field"])
	assert.Equal(t, "2025-09-20 17:00:00", change["old"])
	assert.Equal(t, "2025-09-21 15:00:00", change["new"])
}

func TestSnsPublisher_PublishChanges_WritesTopicArnAndRawJSONMessage(t *testing.T) {
	// Given
	stub := &stubSnsClient{}
	publisher := snsPublisher{client: stub, topicArn: "arn:aws:sns:eu-central-1:123:volley"}
	moved := baseGame()
	changes := []GameChange{{
		Game:    moved,
		Changes: []FieldChange{{Field: FieldPlayDate, Old: "a", New: "b"}},
	}}

	// When
	err := publisher.publishChanges(context.Background(), changes)

	// Then
	assert.Nil(t, err)
	assert.Equal(t, 1, stub.calls)
	if assert.NotNil(t, stub.input) {
		assert.Equal(t, "arn:aws:sns:eu-central-1:123:volley", *stub.input.TopicArn)
		var notification changeNotification
		assert.Nil(t, json.Unmarshal([]byte(*stub.input.Message), &notification))
		assert.Equal(t, "volley.matches.changed", notification.Type)
	}
}

func TestSnsPublisher_PublishChanges_NoOpForEmptyBatch(t *testing.T) {
	// Given
	stub := &stubSnsClient{}
	publisher := snsPublisher{client: stub, topicArn: "arn:aws:sns:eu-central-1:123:volley"}

	// When
	err := publisher.publishChanges(context.Background(), nil)

	// Then
	assert.Nil(t, err)
	assert.Equal(t, 0, stub.calls)
}

func TestSnsPublisher_PublishChanges_ReturnsClientError(t *testing.T) {
	// Given
	stub := &stubSnsClient{err: errors.New("sns is down")}
	publisher := snsPublisher{client: stub, topicArn: "arn:aws:sns:eu-central-1:123:volley"}

	// When
	err := publisher.publishChanges(context.Background(), []GameChange{{Game: baseGame()}})

	// Then
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "sns is down")
	}
}

func TestSnsPublisher_PublishNewGames_UsesDistinctEventAndPayload(t *testing.T) {
	// Given
	stub := &stubSnsClient{}
	publisher := snsPublisher{client: stub, topicArn: "arn:aws:sns:eu-central-1:123:volley"}
	newGame := baseGame()

	// When
	err := publisher.publishNewGames(context.Background(), []Game{newGame})

	// Then
	assert.Nil(t, err)
	assert.Equal(t, 1, stub.calls)
	if assert.NotNil(t, stub.input) {
		message := *stub.input.Message
		assert.Contains(t, message, "volley.matches.new")
		assert.NotContains(t, message, "\"changes\"")
	}
}

func TestSnsPublisher_PublishNewGames_NoOpForEmptyBatch(t *testing.T) {
	// Given
	stub := &stubSnsClient{}
	publisher := snsPublisher{client: stub, topicArn: "arn:aws:sns:eu-central-1:123:volley"}

	// When
	err := publisher.publishNewGames(context.Background(), nil)

	// Then
	assert.Nil(t, err)
	assert.Equal(t, 0, stub.calls)
}
