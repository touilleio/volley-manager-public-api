package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

var (
	mcpNow      = time.Date(2026, 1, 14, 12, 0, 0, 0, time.UTC)
	mcpLocation = time.UTC
)

func seededState(games ...Game) *state {
	s := newState(nil)
	s.rawGames = games
	return s
}

func withResult(game Game, winner string, homeSets int, awaySets int) Game {
	game.ResultSummary.Data.Winner = winner
	game.ResultSummary.Data.WonSetsHomeTeam = homeSets
	game.ResultSummary.Data.WonSetsAwayTeam = awaySets
	return game
}

func TestNextWeekMatches(t *testing.T) {
	next := gameAt("2026-01-20 18:30:00")
	thisWeek := gameAt("2026-01-17 18:30:00")
	farAway := gameAt("2026-02-10 20:00:00")

	output := nextWeekMatches(seededState(next, thisWeek, farAway), mcpNow, mcpLocation, map[string]string{})

	if assert.Len(t, output.Matches, 1) {
		assert.Equal(t, "2026-01-20 18:30:00", output.Matches[0].PlayDate)
		assert.Equal(t, "Gibloux Volley F1", output.Matches[0].HomeTeam)
	}
}

func TestNextWeekMatches_AppliesCaptionReplacement(t *testing.T) {
	next := gameAt("2026-01-20 18:30:00")

	output := nextWeekMatches(seededState(next), mcpNow, mcpLocation,
		map[string]string{"Gibloux Volley F1": "F1"})

	if assert.Len(t, output.Matches, 1) {
		assert.Equal(t, "F1", output.Matches[0].HomeTeam)
	}
}

func TestCurrentWeekResults(t *testing.T) {
	played := withResult(gameAt("2026-01-12 20:00:00"), "home", 3, 1)
	upcoming := gameAt("2026-01-17 18:30:00")
	lastWeek := withResult(gameAt("2026-01-11 20:00:00"), "away", 1, 3)

	output := currentWeekResults(seededState(played, upcoming, lastWeek), mcpNow, mcpLocation, map[string]string{})

	if assert.Len(t, output.Matches, 1) {
		assert.Equal(t, "home", output.Matches[0].Winner)
		assert.Equal(t, 3, output.Matches[0].WonSetsHomeTeam)
		assert.Equal(t, 1, output.Matches[0].WonSetsAwayTeam)
	}
}

// Verifies the whole MCP stack end to end: the streamable HTTP transport
// answers an initialize handshake with this server's identity.
func TestMcpServerInitialize(t *testing.T) {
	server := newMcpServer(seededState(), mcpLocation, map[string]string{})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{JSONResponse: true})
	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	body := `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2025-11-25", "capabilities": {}, "clientInfo": {"name": "test", "version": "0"}}}`
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, testServer.URL, bytes.NewBufferString(body))
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(req)

	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var rpcResponse map[string]json.RawMessage
	assert.Nil(t, json.NewDecoder(resp.Body).Decode(&rpcResponse))
	assert.True(t, strings.Contains(string(rpcResponse["result"]), "volley-manager-public-api"),
		"initialize result should carry the server name, got %s", rpcResponse["result"])
}
