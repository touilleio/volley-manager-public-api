package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Simulates a restart followed by a poll: the detector is primed from a
// snapshot in which one managed game was moved (date + hall), and must
// deliver exactly one Telegram message describing that move.
func TestPollDiffNotifiesTelegram(t *testing.T) {
	raw, err := os.ReadFile("test-data/games-with-cup.json")
	assert.Nil(t, err)
	var allGames []Game
	assert.Nil(t, json.Unmarshal(raw, &allGames))

	theState := newState([]int{6631})
	current := make([]Game, 0)
	for _, game := range allGames {
		if theState.isManagedTeam(game.Teams.Home.TeamId) || theState.isManagedTeam(game.Teams.Away.TeamId) {
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

	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		assert.Nil(t, json.NewDecoder(r.Body).Decode(&payload))
		received <- payload["text"]
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	telegram := newTelegramNotifier("TEST_TOKEN", "12345")
	telegram.apiBase = server.URL
	assert.Nil(t, telegram.notifyChanges(context.Background(), changes))

	message := <-received
	assert.Contains(t, message, "📅")
	assert.Contains(t, message, "📍")
	assert.Contains(t, message, "Ancienne Salle, Fribourg")
	assert.Contains(t, message, current[0].Teams.Home.Caption)
}
