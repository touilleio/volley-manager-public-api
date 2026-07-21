package main

import "time"

const weekDuration = 7 * 24 * time.Hour

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

func getNextWeekGames(games []Game, now time.Time, location *time.Location) []Game {
	return getGamesInWindow(games, now, now.Add(weekDuration), location)
}

func getLastWeekGames(games []Game, now time.Time, location *time.Location) []Game {
	return getGamesInWindow(games, now.Add(-weekDuration), now, location)
}
