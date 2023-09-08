package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

const gamesCollectionUri = "https://api.volleyball.ch/indoor/games"
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
		httpClient: &http.Client{},
	}
	return &internalFetcher, nil
}

func (f fetcher) fetch() error {

	// Fetch rawGames collection
	req, err := http.NewRequest("GET", gamesCollectionUri, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return err
	}

	req.Header.Set("Authorization", f.apiKey)
	resp, err := f.httpClient.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return err
	}

	var games []Game
	if err := json.Unmarshal(body, &games); err != nil {
		fmt.Printf("Error unmarshalling response: %v\n", err)
		return err
	}

	// Fetch the rawRankings
	req, err = http.NewRequest("GET", clubRankingsUri, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return err
	}

	req.Header.Set("Authorization", f.apiKey)
	resp, err = f.httpClient.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return err
	}

	var rankings []GroupRankings
	if err := json.Unmarshal(body, &rankings); err != nil {
		fmt.Printf("Error unmarshalling response: %v\n", err)
		return err
	}

	f.state.rawGames = games
	f.state.rawRankings = rankings

	return nil
}
