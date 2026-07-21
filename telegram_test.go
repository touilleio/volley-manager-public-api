package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func changedGame() []GameChange {
	moved := baseGame()
	moved.PlayDate = "2025-09-21 15:00:00"
	return []GameChange{{
		Game: moved,
		Changes: []FieldChange{
			{Field: FieldPlayDate, Old: "2025-09-20 17:00:00", New: "2025-09-21 15:00:00"},
			{Field: FieldHall, Old: "Salle du Collège, Fribourg", New: "Gymnase, Bulle"},
		},
	}}
}

func TestFormatChangesMessage(t *testing.T) {
	message := formatChangesMessage(changedGame())

	assert.Contains(t, message, "🏐")
	assert.Contains(t, message, "Gibloux Volley F1 vs Volley Fribourg")
	assert.Contains(t, message, "📅 Date : Samedi 20.09.2025 17h00 → Dimanche 21.09.2025 15h00")
	assert.Contains(t, message, "📍 Salle : Salle du Collège, Fribourg → Gymnase, Bulle")
}

func TestFormatChangesMessage_TruncatesLongMessages(t *testing.T) {
	changes := make([]GameChange, 0, 300)
	for i := 0; i < 300; i++ {
		changes = append(changes, changedGame()[0])
	}
	message := formatChangesMessage(changes)
	assert.LessOrEqual(t, len([]rune(message)), telegramMaxMessageLen+2)
}

func TestTelegramNotifier_SendsMessage(t *testing.T) {
	type sendMessageRequest struct {
		ChatID string `json:"chat_id"`
		Text   string `json:"text"`
	}
	received := make(chan sendMessageRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/botTEST_TOKEN/sendMessage", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var payload sendMessageRequest
		assert.Nil(t, json.NewDecoder(r.Body).Decode(&payload))
		received <- payload
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	notifier := newTelegramNotifier("TEST_TOKEN", "12345")
	notifier.apiBase = server.URL

	err := notifier.notifyChanges(context.Background(), changedGame())

	assert.Nil(t, err)
	payload := <-received
	assert.Equal(t, "12345", payload.ChatID)
	assert.Contains(t, payload.Text, "Gibloux Volley F1 vs Volley Fribourg")
}

func TestTelegramNotifier_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"ok":false,"description":"Unauthorized"}`))
	}))
	defer server.Close()

	notifier := newTelegramNotifier("BAD_TOKEN", "12345")
	notifier.apiBase = server.URL

	err := notifier.notifyChanges(context.Background(), changedGame())

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "401")
	}
}

func TestTelegramNotifier_NoChangesSkipsCall(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	notifier := newTelegramNotifier("TEST_TOKEN", "12345")
	notifier.apiBase = server.URL

	assert.Nil(t, notifier.notifyChanges(context.Background(), nil))
	assert.False(t, called)
}

func TestLogNotifier(t *testing.T) {
	var buf bytes.Buffer
	notifier := newLogNotifier(&buf)

	assert.Nil(t, notifier.notifyChanges(context.Background(), changedGame()))
	assert.Contains(t, buf.String(), "Gibloux Volley F1 vs Volley Fribourg")

	buf.Reset()
	assert.Nil(t, notifier.notifyChanges(context.Background(), nil))
	assert.Empty(t, buf.String())
}

func TestTruncateRunes_KeepsMultibyteIntact(t *testing.T) {
	truncated := truncateRunes(strings.Repeat("🏐", 10), 5)
	assert.Equal(t, strings.Repeat("🏐", 5)+"\n…", truncated)
	assert.Equal(t, "abc", truncateRunes("abc", 5))
}
