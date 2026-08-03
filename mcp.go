package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/touilleio/volley-manager-public-api/internal/buildinfo"
)

// newMcpHandler serves the read-only tools statelessly: without
// Stateless, every unauthenticated initialize is retained in memory until
// process exit, which is an unbounded resource-exhaustion vector.
func newMcpHandler(s *state, presenter gamePresenter) http.Handler {
	server := newMcpServer(s, presenter)
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{JSONResponse: true, Stateless: true})
}

type matchesOutput struct {
	Matches []GamePublic `json:"matches" jsonschema:"the club matches in the requested time window"`
}

type teamOutput struct {
	TeamID  int    `json:"teamId" jsonschema:"the team identifier to pass to get_upcoming_matches_for_team"`
	Caption string `json:"caption" jsonschema:"the team's display caption"`
}

type teamsOutput struct {
	Teams []teamOutput `json:"teams" jsonschema:"all teams managed by this server"`
}

type teamMatchesInput struct {
	TeamID int `json:"teamId" jsonschema:"the team identifier returned by list_all_teams"`
}

type noInput struct{}

func newMcpServer(s *state, presenter gamePresenter) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "volley-manager-public-api",
		Version: buildinfo.Version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_next_week_upcoming_matches",
		Description: "Get the club's upcoming matches of next week (Monday to Sunday)",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, matchesOutput, error) {
		return nil, nextWeekMatches(s, time.Now(), presenter), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_current_week_match_results",
		Description: "Get the club's match results of the current week (from Monday up to now)",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, matchesOutput, error) {
		return nil, currentWeekResults(s, time.Now(), presenter), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_all_teams",
		Description: "List all managed teams and their teamId values",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, teamsOutput, error) {
		return nil, allTeams(s, presenter), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_upcoming_matches_for_team",
		Description: "Get all upcoming matches for a teamId returned by list_all_teams",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input teamMatchesInput) (*mcp.CallToolResult, matchesOutput, error) {
		return teamUpcomingMatches(s, presenter, input)
	})

	return server
}

func nextWeekMatches(s *state, now time.Time, presenter gamePresenter) matchesOutput {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return matchesOutput{Matches: presenter.toGamesPublic(getNextWeekGames(s.rawGames, now, presenter.location))}
}

func currentWeekResults(s *state, now time.Time, presenter gamePresenter) matchesOutput {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return matchesOutput{Matches: presenter.toGamesPublic(getCurrentWeekGames(s.rawGames, now, presenter.location))}
}

func allTeams(s *state, presenter gamePresenter) teamsOutput {
	s.lock.RLock()
	defer s.lock.RUnlock()

	teams := make([]teamOutput, 0, len(s.teams))
	for teamID, team := range s.teams {
		caption := team.Caption
		if replacement, ok := presenter.teamCaptionReplacements[caption]; ok {
			caption = replacement
		}
		teams = append(teams, teamOutput{TeamID: teamID, Caption: caption})
	}
	sort.Slice(teams, func(i, j int) bool {
		return teams[i].TeamID < teams[j].TeamID
	})
	return teamsOutput{Teams: teams}
}

func teamUpcomingMatches(s *state, presenter gamePresenter, input teamMatchesInput) (*mcp.CallToolResult, matchesOutput, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if _, ok := s.teams[input.TeamID]; !ok {
		result := &mcp.CallToolResult{}
		result.SetError(fmt.Errorf("unknown teamId %d", input.TeamID))
		return result, matchesOutput{}, nil
	}
	return nil, matchesOutput{Matches: presenter.toUpcomingGamesPublic(s.gamesPerTeam[input.TeamID])}, nil
}
