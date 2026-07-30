package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
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
	gamesPublic := a.presenter().toUpcomingGamesPublic(a.state.rawGames)
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
	gamesPublic := a.presenter().toUpcomingGamesPublic(a.state.gamesPerTeam[teamId])
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
	icsEncoded := a.presenter().toIcal(upcomingGames)

	if team, ok := a.state.teams[teamId]; ok {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", team.Caption))
	}
	c.Data(http.StatusOK, "text/calendar", []byte(icsEncoded))
}

func (a *api) pastGames(c *gin.Context) {
	gamesPublic := a.presenter().toPastGamesPublic(a.state.rawGames)
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
	gamesPublic := a.presenter().toPastGamesPublic(a.state.gamesPerTeam[teamId])
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

	mcpServer := newMcpServer(a.state, a.presenter())
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{JSONResponse: true})
	r.Any("/mcp", gin.WrapH(mcpHandler))

	g.Go(func() error {
		err := r.Run(address)
		if err != nil {
			slog.Error("Got an error", "err", err)
		}
		return err
	})
}

func (a *api) presenter() gamePresenter {
	return newGamePresenter(a.location, a.teamCaptionReplacement, a.state.isCup)
}
