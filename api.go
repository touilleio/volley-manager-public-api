package main

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"net/http"
	"strconv"
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

func (a *api) upcoming(c *gin.Context) {
	games := a.state.rawGames
	c.JSON(http.StatusOK, games)
}

func (a *api) teamUpcoming(c *gin.Context) {
	a.state.lock.RLock()
	defer a.state.lock.RUnlock()
	teamIdStr := c.Param("teamid")
	teamId, err := strconv.Atoi(teamIdStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid teamId %s, err = %s", teamIdStr, err.Error())
		return
	}
	games := a.state.gamesPerTeam[teamId]
	c.JSON(http.StatusOK, games)
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

func (a api) run(address string, g *errgroup.Group) {

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.GET("/upcoming", a.upcoming)
	r.GET("/upcoming/:teamid", a.teamUpcoming)
	r.GET("/ranking/:teamid", a.teamRanking)

	g.Go(func() error {
		err := r.Run(address)
		if err != nil {
			log.WithError(err).Errorf("Got an error")
		}
		return err
	})
}
