package main

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/touilleio/volley-manager-public-api/internal/buildinfo"
)

type matchesOutput struct {
	Matches []GamePublic `json:"matches" jsonschema:"the club matches in the requested time window"`
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
