package main

import (
	"fmt"
	ics "github.com/arran4/golang-ical"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type api struct {
	state                  *state
	teamCaptionReplacement map[string]string
	location               *time.Location
}

func newApi(state *state, teamCaptionReplacementList []string) *api {

	location, err := time.LoadLocation(timezone)
	if err != nil {
		panic(err)
	}

	teamCaptionReplacement := make(map[string]string)
	for _, t := range teamCaptionReplacementList {
		parts := strings.Split(t, ":")
		if len(parts) >= 2 {
			teamCaptionReplacement[parts[0]] = parts[1]
		}
	}

	api := api{
		state:                  state,
		teamCaptionReplacement: teamCaptionReplacement,
		location:               location,
	}
	return &api
}

const (
	timeFormat       = "2006-01-02 15:04:05"
	outputTimeFormat = "Monday 02.01.2006 15h04"
	timezone         = "Europe/Zurich"
)

func (a *api) upcomingGames(c *gin.Context) {
	gamesPublic := toUpcomingGamesPublic(a.state.rawGames, a.location, a.teamCaptionReplacement)
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
	gamesPublic := toUpcomingGamesPublic(a.state.gamesPerTeam[teamId], a.location, a.teamCaptionReplacement)
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
	upcomingGames := getUpcomingGames(a.state.gamesPerTeam[teamId], a.location)
	icsEncoded := toIcal(upcomingGames, a.location)

	if team, ok := a.state.teams[teamId]; ok {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", team.Caption))
	}
	c.Data(http.StatusOK, "text/calendar", []byte(icsEncoded))
}

func (a *api) pastGames(c *gin.Context) {
	gamesPublic := toPastGamesPublic(a.state.rawGames, a.location, a.teamCaptionReplacement)
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
	gamesPublic := toPastGamesPublic(a.state.gamesPerTeam[teamId], a.location, a.teamCaptionReplacement)
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

	dedupTeamRanking := make([]TeamRanking, 0, len(rankings.Ranking))
	teams := make(map[string]bool)

	for _, r := range rankings.Ranking {
		_, ok := teams[r.TeamCaption]
		if !ok {
			teams[r.TeamCaption] = true
			if replacement, ok := a.teamCaptionReplacement[r.TeamCaption]; ok {
				r.TeamCaption = replacement
			}
			dedupTeamRanking = append(dedupTeamRanking, r)
		}
	}
	c.JSON(http.StatusOK, dedupTeamRanking)
}

func (a *api) teams(c *gin.Context) {
	a.state.lock.RLock()
	defer a.state.lock.RUnlock()
	teams := make(map[int]string, len(a.state.teams))
	for k, v := range a.state.teams {
		if replacement, ok := a.teamCaptionReplacement[v.Caption]; ok {
			teams[k] = replacement
		} else {
			teams[k] = v.Caption
		}
	}
	c.JSON(http.StatusOK, teams)
}

func (a api) run(address string, g *errgroup.Group) {

	r := gin.Default()
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

	r.Static("/static", "/static")
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static")
	})

	g.Go(func() error {
		err := r.Run(address)
		if err != nil {
			log.WithError(err).Errorf("Got an error")
		}
		return err
	})
}

var daysInFrench = strings.NewReplacer(
	"Monday", "Lundi",
	"Tuesday", "Mardi",
	"Wednesday", "Mercredi",
	"Thursday", "Jeudi",
	"Friday", "Vendredi",
	"Saturday", "Samedi",
	"Sunday", "Dimanche")

func toGamePublic(game Game, location *time.Location, teamCaptionReplacements map[string]string) GamePublic {
	//parsedTime, _ := time.ParseInLocation(timeFormat, game.PlayDate, location)
	homeTeam := game.Teams.Home.Caption
	if replacement, ok := teamCaptionReplacements[homeTeam]; ok {
		homeTeam = replacement
	}
	awayTeam := game.Teams.Away.Caption
	if replacement, ok := teamCaptionReplacements[awayTeam]; ok {
		awayTeam = replacement
	}
	gp := GamePublic{
		PlayDate: game.PlayDate, //daysInFrench.Replace(parsedTime.In(location).Format(outputTimeFormat)),
		HomeTeam: homeTeam,
		AwayTeam: awayTeam,
		League:   game.League.Caption,
		Hall:     fmt.Sprintf("%s, %s", game.Hall.Caption, game.Hall.City),
	}
	if game.ResultSummary.Data.Winner != "" {
		gp.Winner = game.ResultSummary.Data.Winner
		gp.WonSetsAwayTeam = game.ResultSummary.Data.WonSetsAwayTeam
		gp.WonSetsHomeTeam = game.ResultSummary.Data.WonSetsHomeTeam
	}
	return gp
}

func getUpcomingGames(games []Game, location *time.Location) []Game {
	upcomingGames := make([]Game, 0, len(games))
	for _, g := range games {
		parsedTime, err := time.ParseInLocation(timeFormat, g.PlayDate, location)
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

func getPastGames(games []Game, location *time.Location) []Game {
	upcomingGames := make([]Game, 0, len(games))
	for _, g := range games {
		parsedTime, err := time.ParseInLocation(timeFormat, g.PlayDate, location)
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

func toUpcomingGamesPublic(games []Game, location *time.Location, teamCaptionReplacements map[string]string) []GamePublic {
	gamesPublic := make([]GamePublic, 0, len(games))
	for _, g := range getUpcomingGames(games, location) {
		gamesPublic = append(gamesPublic, toGamePublic(g, location, teamCaptionReplacements))
	}
	return gamesPublic
}

func toPastGamesPublic(games []Game, location *time.Location, teamCaptionReplacements map[string]string) []GamePublic {
	gamesPublic := make([]GamePublic, 0, len(games))
	for _, g := range getPastGames(games, location) {
		gamesPublic = append(gamesPublic, toGamePublic(g, location, teamCaptionReplacements))
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

func toIcal(games []Game, location *time.Location) string {

	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodRequest)

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
