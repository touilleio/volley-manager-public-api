package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	windowNow      = time.Date(2026, 1, 14, 12, 0, 0, 0, time.UTC)
	windowLocation = time.UTC
)

func gameAt(playDate string) Game {
	game := baseGame()
	game.PlayDate = playDate
	return game
}

func TestGetNextWeekGames(t *testing.T) {
	games := []Game{
		gameAt("2026-01-13 20:00:00"), // yesterday: out
		gameAt("2026-01-14 12:00:00"), // right now: in
		gameAt("2026-01-17 18:30:00"), // in 3 days: in
		gameAt("2026-01-21 11:59:00"), // just before the end: in
		gameAt("2026-01-21 12:00:00"), // exactly now+7d: out
		gameAt("2026-02-10 20:00:00"), // far future: out
		gameAt("not a date"),          // unparseable: out
	}

	nextWeek := getNextWeekGames(games, windowNow, windowLocation)

	if assert.Len(t, nextWeek, 3) {
		assert.Equal(t, "2026-01-14 12:00:00", nextWeek[0].PlayDate)
		assert.Equal(t, "2026-01-17 18:30:00", nextWeek[1].PlayDate)
		assert.Equal(t, "2026-01-21 11:59:00", nextWeek[2].PlayDate)
	}
}

func TestGetLastWeekGames(t *testing.T) {
	games := []Game{
		gameAt("2026-01-07 12:00:00"), // exactly now-7d: in
		gameAt("2026-01-10 20:00:00"), // 4 days ago: in
		gameAt("2026-01-14 11:59:00"), // just before now: in
		gameAt("2026-01-14 12:00:00"), // right now: out
		gameAt("2026-01-06 20:00:00"), // 8 days ago: out
		gameAt("2026-01-20 20:00:00"), // future: out
		gameAt(""),                    // unparseable: out
	}

	lastWeek := getLastWeekGames(games, windowNow, windowLocation)

	if assert.Len(t, lastWeek, 3) {
		assert.Equal(t, "2026-01-07 12:00:00", lastWeek[0].PlayDate)
		assert.Equal(t, "2026-01-10 20:00:00", lastWeek[1].PlayDate)
		assert.Equal(t, "2026-01-14 11:59:00", lastWeek[2].PlayDate)
	}
}

func TestGetGamesInWindow_Empty(t *testing.T) {
	assert.Empty(t, getNextWeekGames(nil, windowNow, windowLocation))
	assert.Empty(t, getLastWeekGames(nil, windowNow, windowLocation))
}
