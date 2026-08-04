package main

import (
	"fmt"
	"time"

	ics "github.com/arran4/golang-ical"
)

type gamePresenter struct {
	location                *time.Location
	teamCaptionReplacements map[string]string
	isCup                   func(Game) bool
}

func newGamePresenter(location *time.Location, teamCaptionReplacements map[string]string, isCup func(Game) bool) gamePresenter {
	return gamePresenter{
		location:                location,
		teamCaptionReplacements: teamCaptionReplacements,
		isCup:                   isCup,
	}
}

func (p gamePresenter) toGamePublic(game Game) GamePublic {
	homeTeam := game.Teams.Home.Caption
	if replacement, ok := p.teamCaptionReplacements[homeTeam]; ok {
		homeTeam = replacement
	}
	awayTeam := game.Teams.Away.Caption
	if replacement, ok := p.teamCaptionReplacements[awayTeam]; ok {
		awayTeam = replacement
	}
	isCup := false
	if p.isCup != nil {
		isCup = p.isCup(game)
	}
	public := GamePublic{
		GameId:   game.GameId,
		PlayDate: game.PlayDate,
		HomeTeam: homeTeam,
		AwayTeam: awayTeam,
		League:   game.League.Caption,
		Hall:     fmt.Sprintf("%s, %s", game.Hall.Caption, game.Hall.City),
		IsCup:    isCup,
	}
	if game.ResultSummary.Data.Winner != "" {
		public.Winner = game.ResultSummary.Data.Winner
		public.WonSetsAwayTeam = game.ResultSummary.Data.WonSetsAwayTeam
		public.WonSetsHomeTeam = game.ResultSummary.Data.WonSetsHomeTeam
	}
	return public
}

func (p gamePresenter) toGamesPublic(games []Game) []GamePublic {
	gamesPublic := make([]GamePublic, 0, len(games))
	for _, game := range games {
		gamesPublic = append(gamesPublic, p.toGamePublic(game))
	}
	return gamesPublic
}

func (p gamePresenter) toUpcomingGamesPublic(games []Game) []GamePublic {
	return p.toGamesPublic(getUpcomingGames(games, p.location))
}

func (p gamePresenter) toPastGamesPublic(games []Game) []GamePublic {
	return p.toGamesPublic(getPastGames(games, p.location))
}

func getUpcomingGames(games []Game, location *time.Location) []Game {
	upcomingGames := make([]Game, 0, len(games))
	for _, game := range games {
		parsedTime, err := time.ParseInLocation(timeFormat, game.PlayDate, location)
		if err == nil && parsedTime.After(time.Now()) {
			upcomingGames = append(upcomingGames, game)
		}
	}
	return upcomingGames
}

func getPastGames(games []Game, location *time.Location) []Game {
	pastGames := make([]Game, 0, len(games))
	for _, game := range games {
		parsedTime, err := time.ParseInLocation(timeFormat, game.PlayDate, location)
		if err == nil && time.Now().After(parsedTime) {
			pastGames = append(pastGames, game)
		}
	}
	return pastGames
}

type GamePublic struct {
	GameId          int    `json:"gameId"`
	PlayDate        string `json:"playDate"`
	HomeTeam        string `json:"homeTeam"`
	AwayTeam        string `json:"awayTeam"`
	League          string `json:"phase"`
	Hall            string `json:"hall"`
	WonSetsHomeTeam int    `json:"wonSetsHomeTeam"`
	WonSetsAwayTeam int    `json:"wonSetsAwayTeam"`
	Winner          string `json:"winner"`
	IsCup           bool   `json:"isCup"`
}

func (p gamePresenter) toIcal(games []Game) string {
	calendar := ics.NewCalendar()
	calendar.SetMethod(ics.MethodRequest)

	for _, game := range games {
		event := calendar.AddEvent(fmt.Sprintf("sv-%d", game.GameId))
		parsedTime, err := time.ParseInLocation(timeFormat, game.PlayDate, p.location)
		if err != nil {
			continue
		}
		event.SetCreatedTime(time.Now())
		event.SetClass(ics.ClassificationPublic)
		event.SetDtStampTime(parsedTime)
		event.SetModifiedAt(time.Now())
		event.SetStartAt(parsedTime)
		event.SetEndAt(parsedTime.Add(2 * time.Hour))
		event.SetSummary(fmt.Sprintf("Match %s vs %s", game.Teams.Home.Caption, game.Teams.Away.Caption))
		event.SetLocation(fmt.Sprintf("%s, %s", game.Hall.Caption, game.Hall.City))
		event.SetDescription(fmt.Sprintf("Match %s, %s vs %s, %s @ %s %s", game.League.Caption, game.Teams.Home.Caption, game.Teams.Away.Caption, parsedTime, game.Hall.Caption, game.Hall.City))
	}
	return calendar.Serialize()
}
