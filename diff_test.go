package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var diffNow = time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)

func makeGame(gameId int, playDate string, hallId int, hallCaption string, hallCity string, homeId int, homeCaption string, awayId int, awayCaption string, status int) Game {
	game := Game{
		GameId:   gameId,
		PlayDate: playDate,
		Status:   status,
		Hall: Hall{
			HallId:  hallId,
			Caption: hallCaption,
			City:    hallCity,
		},
	}
	game.Teams.Home = Team{TeamId: homeId, Caption: homeCaption}
	game.Teams.Away = Team{TeamId: awayId, Caption: awayCaption}
	return game
}

func baseGame() Game {
	return makeGame(1, "2025-09-20 17:00:00", 10, "Salle du Collège", "Fribourg",
		100, "Gibloux Volley F1", 200, "Volley Fribourg", 1)
}

func TestDiffGames_NoChange(t *testing.T) {
	previous := []Game{baseGame()}
	current := []Game{baseGame()}
	assert.Empty(t, diffGames(previous, current, diffNow))
}

func TestDiffGames_PlayDateChanged(t *testing.T) {
	previous := []Game{baseGame()}
	moved := baseGame()
	moved.PlayDate = "2025-09-21 15:00:00"

	changes := diffGames(previous, []Game{moved}, diffNow)

	if assert.Len(t, changes, 1) && assert.Len(t, changes[0].Changes, 1) {
		assert.Equal(t, FieldPlayDate, changes[0].Changes[0].Field)
		assert.Equal(t, "2025-09-20 17:00:00", changes[0].Changes[0].Old)
		assert.Equal(t, "2025-09-21 15:00:00", changes[0].Changes[0].New)
	}
}

func TestDiffGames_HallChanged(t *testing.T) {
	previous := []Game{baseGame()}
	moved := baseGame()
	moved.Hall = Hall{HallId: 20, Caption: "Gymnase", City: "Bulle"}

	changes := diffGames(previous, []Game{moved}, diffNow)

	if assert.Len(t, changes, 1) && assert.Len(t, changes[0].Changes, 1) {
		assert.Equal(t, FieldHall, changes[0].Changes[0].Field)
		assert.Equal(t, "Salle du Collège, Fribourg", changes[0].Changes[0].Old)
		assert.Equal(t, "Gymnase, Bulle", changes[0].Changes[0].New)
	}
}

func TestDiffGames_HomeAwaySwapped(t *testing.T) {
	previous := []Game{baseGame()}
	moved := baseGame()
	moved.Teams.Home = Team{TeamId: 200, Caption: "Volley Fribourg"}
	moved.Teams.Away = Team{TeamId: 100, Caption: "Gibloux Volley F1"}

	changes := diffGames(previous, []Game{moved}, diffNow)

	if assert.Len(t, changes, 1) && assert.Len(t, changes[0].Changes, 2) {
		assert.Equal(t, FieldHomeTeam, changes[0].Changes[0].Field)
		assert.Equal(t, FieldAwayTeam, changes[0].Changes[1].Field)
	}
}

func TestDiffGames_StatusChanged(t *testing.T) {
	previous := []Game{baseGame()}
	moved := baseGame()
	moved.Status = 4

	changes := diffGames(previous, []Game{moved}, diffNow)

	if assert.Len(t, changes, 1) && assert.Len(t, changes[0].Changes, 1) {
		assert.Equal(t, FieldStatus, changes[0].Changes[0].Field)
		assert.Equal(t, "1", changes[0].Changes[0].Old)
		assert.Equal(t, "4", changes[0].Changes[0].New)
	}
}

func TestDiffGames_NewAndRemovedGamesIgnored(t *testing.T) {
	previous := []Game{baseGame()}
	current := []Game{
		baseGame(),
		makeGame(2, "2025-09-21 18:00:00", 10, "Salle du Collège", "Fribourg",
			100, "Gibloux Volley F1", 201, "Volley Lausanne", 1),
	}
	assert.Empty(t, diffGames(previous, current, diffNow))
	assert.Empty(t, diffGames(current, previous, diffNow))
}

func TestDiffGames_PastGameIgnored(t *testing.T) {
	previous := []Game{baseGame()}
	moved := baseGame()
	moved.PlayDate = "2025-08-15 20:00:00" // before diffNow

	assert.Empty(t, diffGames(previous, []Game{moved}, diffNow))
}

func TestDiffGames_UnparseablePlayDateStillCompared(t *testing.T) {
	previous := []Game{baseGame()}
	moved := baseGame()
	moved.PlayDate = "to be defined"

	changes := diffGames(previous, []Game{moved}, diffNow)
	assert.Len(t, changes, 1)
}

func TestDiffGames_MultipleGamesAndFields(t *testing.T) {
	game2 := makeGame(2, "2025-09-22 20:00:00", 30, "Salle A", "Estavayer",
		100, "Gibloux Volley F1", 202, "Volley Bern", 1)
	previous := []Game{baseGame(), game2}

	moved1 := baseGame()
	moved1.PlayDate = "2025-09-21 10:00:00"
	moved1.Hall = Hall{HallId: 11, Caption: "Salle B", City: "Fribourg"}
	moved2 := game2
	moved2.Status = 2

	changes := diffGames(previous, []Game{moved1, moved2}, diffNow)

	if assert.Len(t, changes, 2) {
		assert.Len(t, changes[0].Changes, 2)
		assert.Len(t, changes[1].Changes, 1)
	}
}

func TestChangeDetector_FreshFirstPollIsSilent(t *testing.T) {
	// Given
	detector := newChangeDetector(nil)

	// When
	diff := detector.diff([]Game{baseGame()}, diffNow)

	// Then
	assert.Empty(t, diff.Changes)
	assert.Empty(t, diff.NewGames)
}

func TestChangeDetector_PrimedWithSnapshot(t *testing.T) {
	// Given
	detector := newChangeDetector([]Game{baseGame()})

	moved := baseGame()
	moved.PlayDate = "2025-09-21 15:00:00"

	// When
	diff := detector.diff([]Game{moved}, diffNow)

	// Then
	assert.Len(t, diff.Changes, 1)
	assert.Empty(t, diff.NewGames)
}

func TestChangeDetector_SnapshotPrimedNewGamesAreDetected(t *testing.T) {
	// Given
	detector := newChangeDetector([]Game{baseGame()})
	newGame := makeGame(2, "2025-09-21 18:00:00", 10, "Salle du Collège", "Fribourg",
		100, "Gibloux Volley F1", 201, "Volley Lausanne", 1)

	// When
	diff := detector.diff([]Game{baseGame(), newGame}, diffNow)

	// Then
	assert.Empty(t, diff.Changes)
	assert.Len(t, diff.NewGames, 1)
}
