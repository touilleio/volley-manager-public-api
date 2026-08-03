package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	mcpNow      = time.Date(2026, 1, 14, 12, 0, 0, 0, time.UTC)
	mcpLocation = time.UTC
)

func seededState(games ...Game) *state {
	s := newState("", nil, nil)
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

	s := seededState(next, thisWeek, farAway)
	output := nextWeekMatches(s, mcpNow, newGamePresenter(mcpLocation, map[string]string{}, s.isCup))

	if assert.Len(t, output.Matches, 1) {
		assert.Equal(t, "2026-01-20 18:30:00", output.Matches[0].PlayDate)
		assert.Equal(t, "Gibloux Volley F1", output.Matches[0].HomeTeam)
	}
}

func TestNextWeekMatches_AppliesCaptionReplacement(t *testing.T) {
	next := gameAt("2026-01-20 18:30:00")

	s := seededState(next)
	output := nextWeekMatches(s, mcpNow,
		newGamePresenter(mcpLocation, map[string]string{"Gibloux Volley F1": "F1"}, s.isCup))

	if assert.Len(t, output.Matches, 1) {
		assert.Equal(t, "F1", output.Matches[0].HomeTeam)
	}
}

func TestCurrentWeekResults(t *testing.T) {
	played := withResult(gameAt("2026-01-12 20:00:00"), "home", 3, 1)
	upcoming := gameAt("2026-01-17 18:30:00")
	lastWeek := withResult(gameAt("2026-01-11 20:00:00"), "away", 1, 3)

	s := seededState(played, upcoming, lastWeek)
	output := currentWeekResults(s, mcpNow, newGamePresenter(mcpLocation, map[string]string{}, s.isCup))

	if assert.Len(t, output.Matches, 1) {
		assert.Equal(t, "home", output.Matches[0].Winner)
		assert.Equal(t, 3, output.Matches[0].WonSetsHomeTeam)
		assert.Equal(t, 1, output.Matches[0].WonSetsAwayTeam)
	}
}

func TestAllTeams_returns_sorted_teams_with_caption_replacements(t *testing.T) {
	// Given
	s := newState("", nil, nil)
	s.teams = map[int]Team{
		20: {TeamId: 20, Caption: "Second team"},
		10: {TeamId: 10, Caption: "First team"},
	}
	presenter := newGamePresenter(mcpLocation, map[string]string{"First team": "F1"}, s.isCup)

	// When
	output := allTeams(s, presenter)

	// Then
	require.Equal(t, []teamOutput{
		{TeamID: 10, Caption: "F1"},
		{TeamID: 20, Caption: "Second team"},
	}, output.Teams)
}

func TestTeamUpcomingMatches_returns_future_matches_for_the_requested_team(t *testing.T) {
	// Given
	s := newState("", nil, nil)
	s.teams = map[int]Team{10: {TeamId: 10, Caption: "F1"}, 20: {TeamId: 20, Caption: "F2"}}
	future := gameAt("2099-01-20 18:30:00")
	past := gameAt("2000-01-20 18:30:00")
	otherTeam := gameAt("2099-02-20 18:30:00")
	s.gamesPerTeam = map[int][]Game{10: {future, past}, 20: {otherTeam}}
	presenter := newGamePresenter(mcpLocation, map[string]string{"Gibloux Volley F1": "F1"}, s.isCup)

	// When
	result, output, err := teamUpcomingMatches(s, presenter, teamMatchesInput{TeamID: 10})

	// Then
	require.NoError(t, err)
	require.Nil(t, result)
	require.Len(t, output.Matches, 1)
	assert.Equal(t, future.PlayDate, output.Matches[0].PlayDate)
	assert.Equal(t, "F1", output.Matches[0].HomeTeam)
}

func TestTeamUpcomingMatches_returns_tool_error_for_unknown_team(t *testing.T) {
	// Given
	s := newState("", nil, nil)
	presenter := newGamePresenter(mcpLocation, nil, s.isCup)

	// When
	result, output, err := teamUpcomingMatches(s, presenter, teamMatchesInput{TeamID: 999})

	// Then
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.ErrorContains(t, result.GetError(), "unknown teamId 999")
	assert.Empty(t, output.Matches)
}

func TestMcpServer_negotiates_latest_protocol_and_calls_team_tools(t *testing.T) {
	// Given
	s := seededState()
	s.teams = map[int]Team{10: {TeamId: 10, Caption: "Gibloux Volley F1"}}
	s.gamesPerTeam = map[int][]Game{10: {gameAt("2099-01-20 18:30:00")}}
	testServer := httptest.NewServer(newMcpHandler(s,
		newGamePresenter(mcpLocation, map[string]string{"Gibloux Volley F1": "F1"}, s.isCup)))
	defer testServer.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"},
		&mcp.ClientOptions{Capabilities: &mcp.ClientCapabilities{}})

	// When
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:             testServer.URL,
		DisableStandaloneSSE: true,
	}, nil)
	require.NoError(t, err)
	defer session.Close()

	// Then
	assert.Equal(t, "2026-07-28", session.InitializeResult().ProtocolVersion)
	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)
	toolNames := make(map[string]bool, len(tools.Tools))
	for _, tool := range tools.Tools {
		toolNames[tool.Name] = true
	}
	assert.True(t, toolNames["list_all_teams"])
	assert.True(t, toolNames["get_upcoming_matches_for_team"])

	listed, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_all_teams"})
	require.NoError(t, err)
	var teams teamsOutput
	decodeStructuredContent(t, listed.StructuredContent, &teams)
	require.Equal(t, []teamOutput{{TeamID: 10, Caption: "F1"}}, teams.Teams)

	upcoming, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_upcoming_matches_for_team",
		Arguments: teamMatchesInput{TeamID: teams.Teams[0].TeamID},
	})
	require.NoError(t, err)
	var matches matchesOutput
	decodeStructuredContent(t, upcoming.StructuredContent, &matches)
	require.Len(t, matches.Matches, 1)
	assert.Equal(t, "F1", matches.Matches[0].HomeTeam)
}

func decodeStructuredContent(t *testing.T, content any, output any) {
	t.Helper()
	data, err := json.Marshal(content)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, output))
}
