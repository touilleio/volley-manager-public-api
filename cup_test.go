package main

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToGamePublicIncludesIsCup(t *testing.T) {
	cupGame := Game{League: League{LeagueCategoryId: 4, Caption: "Cup"}}

	assert.True(t, newGamePresenter(time.UTC, nil, func(Game) bool { return true }).toGamePublic(cupGame).IsCup)
	assert.False(t, newGamePresenter(time.UTC, nil, func(Game) bool { return false }).toGamePublic(cupGame).IsCup)
}

func TestNextWeekMatchesIncludesCupGames(t *testing.T) {
	cupGame := Game{
		GameId:   42,
		PlayDate: "2026-01-20 18:30:00",
		League:   League{LeagueCategoryId: 4, Caption: "Cup"},
	}
	s := newState("", nil, []int{4})
	s.rawGames = []Game{cupGame}

	output := nextWeekMatches(s, mcpNow, newGamePresenter(mcpLocation, nil, s.isCup))

	if assert.Len(t, output.Matches, 1) {
		assert.True(t, output.Matches[0].IsCup)
	}
}

func TestToIcalIncludesCupGames(t *testing.T) {
	cupGame := Game{
		GameId:   42,
		PlayDate: "2026-01-20 18:30:00",
		League:   League{LeagueCategoryId: 4, Caption: "Cup"},
	}

	calendar := newGamePresenter(time.UTC, nil, nil).toIcal([]Game{cupGame})

	assert.Contains(t, calendar, "UID:sv-"+strconv.Itoa(cupGame.GameId))
}
