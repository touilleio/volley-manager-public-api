package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	// A Wednesday, 12:00 UTC.
	windowNow      = time.Date(2026, 1, 14, 12, 0, 0, 0, time.UTC)
	windowLocation = time.UTC
)

func gameAt(playDate string) Game {
	game := baseGame()
	game.PlayDate = playDate
	return game
}

func TestStartOfWeek(t *testing.T) {
	assert.Equal(t, time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC), startOfWeek(windowNow, windowLocation))
	// Sunday evening belongs to the week started the previous Monday.
	sunday := time.Date(2026, 1, 18, 20, 0, 0, 0, time.UTC)
	assert.Equal(t, time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC), startOfWeek(sunday, windowLocation))
	// Monday 00:00 is its own week's start.
	monday := time.Date(2026, 1, 19, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, monday, startOfWeek(monday, windowLocation))
}

func TestGetNextWeekGames(t *testing.T) {
	games := []Game{
		gameAt("2026-01-17 18:30:00"), // this week (Saturday): out
		gameAt("2026-01-19 00:00:00"), // next Monday 00:00: in
		gameAt("2026-01-20 20:00:00"), // next week: in
		gameAt("2026-01-25 23:59:00"), // next Sunday: in
		gameAt("2026-01-26 00:00:00"), // the Monday after: out
		gameAt("2026-01-12 20:00:00"), // this week: out
		gameAt("not a date"),          // unparseable: out
	}

	nextWeek := getNextWeekGames(games, windowNow, windowLocation)

	if assert.Len(t, nextWeek, 3) {
		assert.Equal(t, "2026-01-19 00:00:00", nextWeek[0].PlayDate)
		assert.Equal(t, "2026-01-20 20:00:00", nextWeek[1].PlayDate)
		assert.Equal(t, "2026-01-25 23:59:00", nextWeek[2].PlayDate)
	}
}

func TestGetNextWeekGames_OnSundayEvening(t *testing.T) {
	// The typical usage: called on Sunday evening, "next week" starts tomorrow.
	sundayEvening := time.Date(2026, 1, 18, 20, 0, 0, 0, time.UTC)
	games := []Game{
		gameAt("2026-01-18 21:00:00"), // later this Sunday: out
		gameAt("2026-01-19 08:00:00"), // tomorrow: in
	}

	nextWeek := getNextWeekGames(games, sundayEvening, windowLocation)

	if assert.Len(t, nextWeek, 1) {
		assert.Equal(t, "2026-01-19 08:00:00", nextWeek[0].PlayDate)
	}
}

func TestGetCurrentWeekGames(t *testing.T) {
	games := []Game{
		gameAt("2026-01-11 23:59:00"), // last Sunday: out
		gameAt("2026-01-12 00:00:00"), // this Monday 00:00: in
		gameAt("2026-01-13 20:00:00"), // yesterday: in
		gameAt("2026-01-14 11:59:00"), // just before now: in
		gameAt("2026-01-14 12:00:00"), // right now: out
		gameAt("2026-01-17 20:00:00"), // this Saturday, not played yet: out
		gameAt(""),                    // unparseable: out
	}

	currentWeek := getCurrentWeekGames(games, windowNow, windowLocation)

	if assert.Len(t, currentWeek, 3) {
		assert.Equal(t, "2026-01-12 00:00:00", currentWeek[0].PlayDate)
		assert.Equal(t, "2026-01-13 20:00:00", currentWeek[1].PlayDate)
		assert.Equal(t, "2026-01-14 11:59:00", currentWeek[2].PlayDate)
	}
}

func TestGetGamesInWindow_Empty(t *testing.T) {
	assert.Empty(t, getNextWeekGames(nil, windowNow, windowLocation))
	assert.Empty(t, getCurrentWeekGames(nil, windowNow, windowLocation))
}
