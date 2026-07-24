package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
)

type stubSqsClient struct {
	input *sqs.SendMessageInput
	err   error
	calls int
}

func (s *stubSqsClient) SendMessage(_ context.Context, params *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	s.calls++
	s.input = params
	return &sqs.SendMessageOutput{}, s.err
}

func TestNewChangeNotification(t *testing.T) {
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

	notification := newChangeNotification(changes, detectedAt)

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

	// The payload keys are part of the consumer contract: they must stay stable.
	raw, err := json.Marshal(notification)
	assert.Nil(t, err)
	var decoded map[string]any
	assert.Nil(t, json.Unmarshal(raw, &decoded))
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

func TestSqsPublisher_Publishes(t *testing.T) {
	stub := &stubSqsClient{}
	publisher := &sqsPublisher{client: stub, queueURL: "https://sqs.eu-central-1.amazonaws.com/123/queue"}

	moved := baseGame()
	err := publisher.publish(context.Background(), []GameChange{{
		Game:    moved,
		Changes: []FieldChange{{Field: FieldPlayDate, Old: "a", New: "b"}},
	}})

	assert.Nil(t, err)
	assert.Equal(t, 1, stub.calls)
	if assert.NotNil(t, stub.input) {
		assert.Equal(t, "https://sqs.eu-central-1.amazonaws.com/123/queue", *stub.input.QueueUrl)
		assert.Contains(t, *stub.input.MessageBody, "\"type\":\"volley.matches.changed\"")
		assert.Contains(t, *stub.input.MessageBody, "Gibloux Volley F1")
	}
}

func TestSqsPublisher_NoChangesSkipsSend(t *testing.T) {
	stub := &stubSqsClient{}
	publisher := &sqsPublisher{client: stub, queueURL: "queue"}

	assert.Nil(t, publisher.publish(context.Background(), nil))
	assert.Equal(t, 0, stub.calls)
}

func TestSqsPublisher_SendError(t *testing.T) {
	stub := &stubSqsClient{err: errors.New("sqs is down")}
	publisher := &sqsPublisher{client: stub, queueURL: "queue"}

	err := publisher.publish(context.Background(), []GameChange{{Game: baseGame()}})

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "sqs is down")
	}
}
