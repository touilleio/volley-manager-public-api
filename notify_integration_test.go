package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Simulates a restart followed by a poll: the detector is primed from a
// snapshot in which one managed game was moved (date + hall), and the
// resulting SQS message must carry that move in its payload.
func TestPollDiffPublishesToSqs(t *testing.T) {
	raw, err := os.ReadFile("test-data/games-with-cup.json")
	assert.Nil(t, err)
	var allGames []Game
	assert.Nil(t, json.Unmarshal(raw, &allGames))

	theState := newState("906295", nil, nil)
	current := make([]Game, 0)
	for _, game := range allGames {
		if theState.isManagedTeam(game.Teams.Home) || theState.isManagedTeam(game.Teams.Away) {
			current = append(current, game)
		}
	}
	if !assert.NotEmpty(t, current) {
		return
	}

	snapshotRaw, err := json.Marshal(current)
	assert.Nil(t, err)
	var previous []Game
	assert.Nil(t, json.Unmarshal(snapshotRaw, &previous))
	previous[0].PlayDate = "2025-09-14 20:45:00"
	previous[0].Hall = Hall{HallId: 999, Caption: "Ancienne Salle", City: "Fribourg"}

	beforeSeason := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	detector := newChangeDetector(previous)
	changes := detector.diff(current, beforeSeason)

	if !assert.Len(t, changes, 1) {
		return
	}
	assert.Equal(t, current[0].GameId, changes[0].Game.GameId)
	assert.Len(t, changes[0].Changes, 2)

	stub := &stubSqsClient{}
	publisher := &sqsPublisher{client: stub, queueURL: "queue"}
	assert.Nil(t, publisher.publish(context.Background(), changes))
	assert.Equal(t, 1, stub.calls)

	var notification changeNotification
	assert.Nil(t, json.Unmarshal([]byte(*stub.input.MessageBody), &notification))
	if assert.Len(t, notification.Games, 1) {
		assert.Equal(t, current[0].GameId, notification.Games[0].GameId)
		assert.Equal(t, current[0].Teams.Home.Caption, notification.Games[0].HomeTeam)
		fields := []string{notification.Games[0].Changes[0].Field, notification.Games[0].Changes[1].Field}
		assert.Contains(t, fields, FieldPlayDate)
		assert.Contains(t, fields, FieldHall)
	}
}
