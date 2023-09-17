package main

import (
	"fmt"
	ics "github.com/arran4/golang-ical"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"net/http"
	"strconv"
	"time"
)

type api struct {
	state *state
}

func newApi(state *state) *api {
	api := api{
		state: state,
	}
	return &api
}

const timeFormat = "2006-01-02 15:04:05"

func (a *api) upcomingGames(c *gin.Context) {
	gamesPublic := toUpcomingGamesPublic(a.state.rawGames)
	c.JSON(http.StatusOK, gamesPublic)
}

func (a *api) teamUpcomingGames(c *gin.Context) {
	a.state.lock.RLock()
	defer a.state.lock.RUnlock()
	teamIdStr := c.Param("teamid")
	teamId, err := strconv.Atoi(teamIdStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid teamId %s, err = %s", teamIdStr, err.Error())
		return
	}
	gamesPublic := toUpcomingGamesPublic(a.state.gamesPerTeam[teamId])
	c.JSON(http.StatusOK, gamesPublic)
}

func (a *api) teamUpcomingGamesICS(c *gin.Context) {
	a.state.lock.RLock()
	defer a.state.lock.RUnlock()
	teamIdStr := c.Param("teamid")
	teamId, err := strconv.Atoi(teamIdStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid teamId %s, err = %s", teamIdStr, err.Error())
		return
	}
	upcomingGames := getUpcomingGames(a.state.gamesPerTeam[teamId])
	icsEncoded := toIcal(upcomingGames)

	c.Data(http.StatusOK, "text/calendar", []byte(icsEncoded))
}

func (a *api) pastGames(c *gin.Context) {
	gamesPublic := toPastGamesPublic(a.state.rawGames)
	c.JSON(http.StatusOK, gamesPublic)
}

func (a *api) teamPastGames(c *gin.Context) {
	a.state.lock.RLock()
	defer a.state.lock.RUnlock()
	teamIdStr := c.Param("teamid")
	teamId, err := strconv.Atoi(teamIdStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid teamId %s, err = %s", teamIdStr, err.Error())
		return
	}
	gamesPublic := toPastGamesPublic(a.state.gamesPerTeam[teamId])
	c.JSON(http.StatusOK, gamesPublic)
}

func (a *api) teamRanking(c *gin.Context) {
	a.state.lock.RLock()
	defer a.state.lock.RUnlock()
	teamIdStr := c.Param("teamid")
	teamId, err := strconv.Atoi(teamIdStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid teamId %s, err = %s", teamIdStr, err.Error())
		return
	}
	rankings := a.state.rankingPerTeam[teamId]
	c.JSON(http.StatusOK, rankings.Ranking)
}

func (a *api) teams(c *gin.Context) {
	a.state.lock.RLock()
	defer a.state.lock.RUnlock()
	teams := make(map[int]string, len(a.state.teams))
	for k, v := range a.state.teams {
		teams[k] = v.Caption
	}
	c.JSON(http.StatusOK, teams)
}

func (a api) run(address string, g *errgroup.Group) {

	r := gin.Default()
	r.StaticFS("/static", http.Dir("/static"))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.GET("/upcoming", a.upcomingGames)
	r.GET("/upcoming/:teamid", a.teamUpcomingGames)
	r.GET("/ics/upcoming/:teamid", a.teamUpcomingGamesICS)
	r.GET("/past", a.pastGames)
	r.GET("/past/:teamid", a.teamPastGames)
	r.GET("/ranking/:teamid", a.teamRanking)
	r.GET("/teams", a.teams)

	g.Go(func() error {
		err := r.Run(address)
		if err != nil {
			log.WithError(err).Errorf("Got an error")
		}
		return err
	})
}

func toGamePublic(game Game) GamePublic {
	gp := GamePublic{
		PlayDate: game.PlayDate,
		HomeTeam: game.Teams.Home.Caption,
		AwayTeam: game.Teams.Away.Caption,
		League:   game.League.Caption,
		Hall:     game.Hall.Caption,
	}
	if game.ResultSummary.Data.Winner != "" {
		gp.Winner = game.ResultSummary.Data.Winner
		gp.WonSetsAwayTeam = game.ResultSummary.Data.WonSetsAwayTeam
		gp.WonSetsHomeTeam = game.ResultSummary.Data.WonSetsHomeTeam
	}
	return gp
}

func getUpcomingGames(games []Game) []Game {
	upcomingGames := make([]Game, 0, len(games))
	for _, g := range games {
		parsedTime, err := time.Parse(timeFormat, g.PlayDate)
		if err != nil {
			// TODO log a warning
			continue
		}
		now := time.Now()
		if parsedTime.After(now) {
			upcomingGames = append(upcomingGames, g)
		}
	}
	return upcomingGames
}

func getPastGames(games []Game) []Game {
	upcomingGames := make([]Game, 0, len(games))
	for _, g := range games {
		parsedTime, err := time.Parse(timeFormat, g.PlayDate)
		if err != nil {
			// TODO log a warning
			continue
		}
		now := time.Now()
		if now.After(parsedTime) {
			upcomingGames = append(upcomingGames, g)
		}
	}
	return upcomingGames
}

func toUpcomingGamesPublic(games []Game) []GamePublic {
	gamesPublic := make([]GamePublic, 0, len(games))
	for _, g := range getUpcomingGames(games) {
		gamesPublic = append(gamesPublic, toGamePublic(g))
	}
	return gamesPublic
}

func toPastGamesPublic(games []Game) []GamePublic {
	gamesPublic := make([]GamePublic, 0, len(games))
	for _, g := range getPastGames(games) {
		gamesPublic = append(gamesPublic, toGamePublic(g))
	}
	return gamesPublic
}

type GamePublic struct {
	PlayDate        string `json:"playDate"`
	HomeTeam        string `json:"homeTeam"`
	AwayTeam        string `json:"awayTeam"`
	League          string `json:"phase"`
	Hall            string `json:"hall"`
	WonSetsHomeTeam int    `json:"wonSetsHomeTeam"`
	WonSetsAwayTeam int    `json:"wonSetsAwayTeam"`
	Winner          string `json:"winner"`
}

func toIcal(games []Game) string {

	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodRequest)
	//cal.SetTimezoneId("Europe/Zurich")
	location, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		panic(err)
	}

	for _, game := range games {
		event := cal.AddEvent(fmt.Sprintf("sv-%d", game.GameId))

		parsedTime, err := time.ParseInLocation(timeFormat, game.PlayDate, location)
		if err != nil {
			// TODO log a warning
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
	return cal.Serialize()
}
