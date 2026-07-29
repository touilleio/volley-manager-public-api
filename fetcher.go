package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

/*
Games Collection
https://api.volleyball.ch/indoor/games

Club Rankings
https://api.volleyball.ch/indoor/ranking

Upcoming Games Collection
https://api.volleyball.ch/indoor/upcomingGames

Recent Results Collection
https://api.volleyball.ch/indoor/recentResults
*/

const gamesCollectionUri = "https://api.volleyball.ch/indoor/games?includeCup=1"
const clubRankingsUri = "https://api.volleyball.ch/indoor/ranking"

type fetcher struct {
	apiKey string
	state  *state
	// Make the request
	httpClient *http.Client
}

func newFetcher(apiKey string, state *state) (*fetcher, error) {
	internalFetcher := fetcher{
		apiKey:     apiKey,
		state:      state,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	return &internalFetcher, nil
}

func (f fetcher) fetch(ctx context.Context) error {

	// Fetch rawGames collection
	req, err := http.NewRequestWithContext(ctx, "GET", gamesCollectionUri, nil)
	if err != nil {
		return fmt.Errorf("creating games request: %w", err)
	}

	req.Header.Set("Authorization", f.apiKey)
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching games: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading games response body: %w", err)
	}

	var games []Game
	if err := json.Unmarshal(body, &games); err != nil {
		return fmt.Errorf("unmarshalling games response: %w (body: %.512s)", err, string(body))
	}

	// Fetch the rawRankings
	req, err = http.NewRequestWithContext(ctx, "GET", clubRankingsUri, nil)
	if err != nil {
		return fmt.Errorf("creating rankings request: %w", err)
	}

	req.Header.Set("Authorization", f.apiKey)
	resp, err = f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching rankings: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading rankings response body: %w", err)
	}

	var rankings []GroupRankings
	if err := json.Unmarshal(body, &rankings); err != nil {
		return fmt.Errorf("unmarshalling rankings response: %w (body: %.512s)", err, string(body))
	}

	f.state.rawGames = games
	f.state.rawRankings = rankings

	return nil
}
