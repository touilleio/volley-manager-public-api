package main

import "time"

// getGamesInWindow returns the games whose play date falls in [from, to).
// Games with an unparseable PlayDate are skipped.
func getGamesInWindow(games []Game, from, to time.Time, location *time.Location) []Game {
	windowGames := make([]Game, 0, len(games))
	for _, g := range games {
		parsedTime, err := time.ParseInLocation(timeFormat, g.PlayDate, location)
		if err != nil {
			continue
		}
		if !parsedTime.Before(from) && parsedTime.Before(to) {
			windowGames = append(windowGames, g)
		}
	}
	return windowGames
}

// startOfWeek returns Monday 00:00 of the week (Monday to Sunday) containing
// t, in the given location. AddDate is used so DST transitions don't shift
// the boundary.
func startOfWeek(t time.Time, location *time.Location) time.Time {
	local := t.In(location)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	daysSinceMonday := (int(local.Weekday()) + 6) % 7
	return midnight.AddDate(0, 0, -daysSinceMonday)
}

// getNextWeekGames returns the games of the next calendar week (Monday 00:00
// to the following Monday 00:00).
func getNextWeekGames(games []Game, now time.Time, location *time.Location) []Game {
	weekStart := startOfWeek(now, location).AddDate(0, 0, 7)
	return getGamesInWindow(games, weekStart, weekStart.AddDate(0, 0, 7), location)
}

// getCurrentWeekGames returns the games of the current calendar week, from
// Monday 00:00 up to now.
func getCurrentWeekGames(games []Game, now time.Time, location *time.Location) []Game {
	return getGamesInWindow(games, startOfWeek(now, location), now, location)
}
