package main

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"net/http"
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

}
func (a *api) ranking(c *gin.Context) {

}

func (a api) run(address string, g *errgroup.Group) {

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.GET("/upcoming", a.upcoming)
	r.GET("/ranking", a.ranking)

	g.Go(func() error {
		err := r.Run(address)
		if err != nil {
			log.WithError(err).Errorf("Got an error")
		}
		return err
	})
}
