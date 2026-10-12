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
const maxCollectionBytes = 32 << 20

type fetcher struct {
	apiKey string
	// Make the request
	httpClient *http.Client
}

func newFetcher(apiKey string) *fetcher {
	return &fetcher{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// fetch downloads the upstream collections without touching shared state. The
// caller is responsible for publishing them atomically once filtered.
func (f fetcher) fetch(ctx context.Context) ([]Game, []GroupRankings, error) {

	// Fetch rawGames collection
	req, err := http.NewRequestWithContext(ctx, "GET", gamesCollectionUri, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("creating games request: %w", err)
	}

	req.Header.Set("Authorization", f.apiKey)
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching games: %w", err)
	}
	defer resp.Body.Close()

	body, err := readCollection(resp)
	if err != nil {
		return nil, nil, fmt.Errorf("reading games response body: %w", err)
	}

	var games []Game
	if err := json.Unmarshal(body, &games); err != nil {
		return nil, nil, fmt.Errorf("unmarshalling games response: %w", err)
	}
	if games == nil {
		return nil, nil, fmt.Errorf("games response must be an array")
	}

	// Fetch the rawRankings
	req, err = http.NewRequestWithContext(ctx, "GET", clubRankingsUri, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("creating rankings request: %w", err)
	}

	req.Header.Set("Authorization", f.apiKey)
	resp, err = f.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching rankings: %w", err)
	}
	defer resp.Body.Close()

	body, err = readCollection(resp)
	if err != nil {
		return nil, nil, fmt.Errorf("reading rankings response body: %w", err)
	}

	var rankings []GroupRankings
	if err := json.Unmarshal(body, &rankings); err != nil {
		return nil, nil, fmt.Errorf("unmarshalling rankings response: %w", err)
	}
	if rankings == nil {
		return nil, nil, fmt.Errorf("rankings response must be an array")
	}

	return games, rankings, nil
}

func readCollection(resp *http.Response) ([]byte, error) {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("upstream returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxCollectionBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxCollectionBytes {
		return nil, fmt.Errorf("upstream collection exceeds %d bytes", maxCollectionBytes)
	}
	return body, nil
}
