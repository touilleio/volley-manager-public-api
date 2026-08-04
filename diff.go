package main

import (
	"fmt"
	"strconv"
	"time"
)

// Field names used in FieldChange.Field
const (
	FieldPlayDate = "playDate"
	FieldHall     = "hall"
	FieldHomeTeam = "homeTeam"
	FieldAwayTeam = "awayTeam"
	FieldStatus   = "status"
)

// FieldChange describes one changed attribute of a game between two polls.
type FieldChange struct {
	Field string `json:"field"`
	Old   string `json:"old"`
	New   string `json:"new"`
}

// GameChange describes all detected changes of one game. Game holds the
// current (new) version of the game, for context in notifications.
type GameChange struct {
	Game    Game
	Changes []FieldChange
}

type GameDiff struct {
	Changes  []GameChange
	NewGames []Game
}

var clubLocation = func() *time.Location {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		panic(err)
	}
	return location
}()

// diffGames compares the previously known games with the freshly polled ones,
// keyed by GameId. Only games present in both lists are compared: new and
// removed games never produce a change (per product decision). Only upcoming
// games are compared, so that post-match status updates on past games do not
// trigger notifications. Games with an unparseable PlayDate are treated as
// upcoming.
func diffGames(previous, current []Game, now time.Time) []GameChange {
	previousById := make(map[int]Game, len(previous))
	for _, g := range previous {
		previousById[g.GameId] = g
	}

	changes := make([]GameChange, 0)
	for _, cur := range current {
		prev, ok := previousById[cur.GameId]
		if !ok {
			continue
		}
		if !isUpcoming(cur, now) {
			continue
		}

		fieldChanges := make([]FieldChange, 0, 4)
		if prev.PlayDate != cur.PlayDate {
			fieldChanges = append(fieldChanges, FieldChange{Field: FieldPlayDate, Old: prev.PlayDate, New: cur.PlayDate})
		}
		if prev.Hall.HallId != cur.Hall.HallId || prev.Hall.Caption != cur.Hall.Caption || prev.Hall.City != cur.Hall.City {
			fieldChanges = append(fieldChanges, FieldChange{Field: FieldHall, Old: hallSummary(prev.Hall), New: hallSummary(cur.Hall)})
		}
		if prev.Teams.Home.TeamId != cur.Teams.Home.TeamId {
			fieldChanges = append(fieldChanges, FieldChange{Field: FieldHomeTeam, Old: prev.Teams.Home.Caption, New: cur.Teams.Home.Caption})
		}
		if prev.Teams.Away.TeamId != cur.Teams.Away.TeamId {
			fieldChanges = append(fieldChanges, FieldChange{Field: FieldAwayTeam, Old: prev.Teams.Away.Caption, New: cur.Teams.Away.Caption})
		}
		// Status is 2 for both scheduled and already-played games in the API feed,
		// so it does not mean "played"; a flip on an upcoming game signals an
		// exceptional state (e.g. postponed/cancelled) worth notifying. Played
		// games never reach this comparison because of the upcoming-only filter.
		if prev.Status != cur.Status {
			fieldChanges = append(fieldChanges, FieldChange{Field: FieldStatus, Old: strconv.Itoa(prev.Status), New: strconv.Itoa(cur.Status)})
		}

		if len(fieldChanges) > 0 {
			changes = append(changes, GameChange{Game: cur, Changes: fieldChanges})
		}
	}
	return changes
}

func newGames(previous, current []Game, now time.Time) []Game {
	previousById := make(map[int]struct{}, len(previous))
	for _, game := range previous {
		previousById[game.GameId] = struct{}{}
	}

	games := make([]Game, 0)
	for _, game := range current {
		if _, ok := previousById[game.GameId]; ok {
			continue
		}
		if !isUpcoming(game, now) {
			continue
		}
		games = append(games, game)
	}
	return games
}

func isUpcoming(game Game, now time.Time) bool {
	parsedTime, err := time.ParseInLocation(timeFormat, game.PlayDate, clubLocation)
	if err != nil {
		return true
	}
	return parsedTime.After(now)
}

func hallSummary(h Hall) string {
	return fmt.Sprintf("%s, %s", h.Caption, h.City)
}

// changeDetector holds the baseline of the last successful poll. It is only
// accessed from the single fetch goroutine in main, so no locking is needed.
type changeDetector struct {
	previous []Game
	primed   bool
}

// newChangeDetector returns a detector primed with the given baseline. A nil
// baseline (fresh install, no snapshot) means the first poll only establishes
// the baseline and never notifies.
func newChangeDetector(previous []Game) *changeDetector {
	return &changeDetector{previous: previous, primed: previous != nil}
}

func (d *changeDetector) diff(current []Game, now time.Time) GameDiff {
	if !d.primed {
		d.previous = current
		d.primed = true
		return GameDiff{}
	}
	diff := GameDiff{
		Changes:  diffGames(d.previous, current, now),
		NewGames: newGames(d.previous, current, now),
	}
	d.previous = current
	return diff
}
