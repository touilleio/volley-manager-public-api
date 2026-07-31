package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// teamIDParam parses the route parameter once and answers with a generic
// client error: internal parse details are never reflected to callers.
func teamIDParam(c *gin.Context) (int, bool) {
	teamID, err := strconv.Atoi(c.Param("teamid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return 0, false
	}
	return teamID, true
}

func (a *api) upcomingGames(c *gin.Context) {
	a.state.lock.RLock()
	defer a.state.lock.RUnlock()
	gamesPublic := a.presenter().toUpcomingGamesPublic(a.state.rawGames)
	c.JSON(http.StatusOK, gamesPublic)
}

func (a *api) teamUpcomingGames(c *gin.Context) {
	a.state.lock.RLock()
	defer a.state.lock.RUnlock()
	teamID, ok := teamIDParam(c)
	if !ok {
		return
	}
	gamesPublic := a.presenter().toUpcomingGamesPublic(a.state.gamesPerTeam[teamID])
	c.JSON(http.StatusOK, gamesPublic)
}

var filenameSanitizer = strings.NewReplacer("\r", "", "\n", "", `"`, "'")

func (a *api) teamUpcomingGamesICS(c *gin.Context) {
	a.state.lock.RLock()
	defer a.state.lock.RUnlock()
	teamID, ok := teamIDParam(c)
	if !ok {
		return
	}
	upcomingGames := getUpcomingGames(a.state.gamesPerTeam[teamID], a.location)
	icsEncoded := a.presenter().toIcal(upcomingGames)

	if team, ok := a.state.teams[teamID]; ok {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filenameSanitizer.Replace(team.Caption)))
	}
	c.Data(http.StatusOK, "text/calendar", []byte(icsEncoded))
}

func (a *api) pastGames(c *gin.Context) {
	a.state.lock.RLock()
	defer a.state.lock.RUnlock()
	gamesPublic := a.presenter().toPastGamesPublic(a.state.rawGames)
	c.JSON(http.StatusOK, gamesPublic)
}

func (a *api) teamPastGames(c *gin.Context) {
	a.state.lock.RLock()
	defer a.state.lock.RUnlock()
	teamID, ok := teamIDParam(c)
	if !ok {
		return
	}
	gamesPublic := a.presenter().toPastGamesPublic(a.state.gamesPerTeam[teamID])
	c.JSON(http.StatusOK, gamesPublic)
}

func (a *api) teamRanking(c *gin.Context) {
	a.state.lock.RLock()
	defer a.state.lock.RUnlock()
	teamID, ok := teamIDParam(c)
	if !ok {
		return
	}
	rankings := a.state.rankingPerTeam[teamID]

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

// securityHeaders applies the response headers that are safe for an embeddable
// read-only API. CSP and frame rules are deployment-specific and documented in
// the README instead of being forced here.
func securityHeaders(c *gin.Context) {
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
	c.Next()
}

func (a *api) router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), securityHeaders)
	// Proxy headers (X-Forwarded-For) are attacker-controlled unless a
	// deployment explicitly trusts its reverse proxies; trust none by default.
	if err := r.SetTrustedProxies(nil); err != nil {
		panic(fmt.Errorf("disabling trusted proxies: %w", err))
	}

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

	r.Any("/mcp", gin.WrapH(newMcpHandler(a.state, a.presenter())))
	return r
}

func (a *api) newHTTPServer(address string) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           a.router(),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func (a *api) run(address string, ctx context.Context, g *errgroup.Group) {
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	server := a.newHTTPServer(address)

	g.Go(func() error {
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	g.Go(func() error {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Got an error while shutting down the HTTP server", "err", err)
			return err
		}
		return nil
	})
}

func (a *api) presenter() gamePresenter {
	return newGamePresenter(a.location, a.teamCaptionReplacement, a.state.isCup)
}
