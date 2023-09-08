package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/sqooba/go-common/logging"
	"github.com/sqooba/go-common/version"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	setLogLevel = flag.String("set-log-level", "", "Change log level. Possible values are trace,debug,info,warn,error,fatal,panic")
)

type EnvConfig struct {
	APIKey           string        `envconfig:"API_KEY"`
	RefreshInterval  time.Duration `envconfig:"REFRESH_INTERVAL" default:"1h"`
	TeamsId          []int         `envconfig:"TEAMS_ID" default:"6631,6632,6633,6634,6635,6636,7681,11902,11903"`
	BindIP           string        `envconfig:"BIND_IP" default:"0.0.0.0"`
	Port             string        `envconfig:"PORT" default:"8080"`
	LogLevel         string        `envconfig:"LOG_LEVEL" default:"debug"`
	MetricsNamespace string        `envconfig:"METRICS_NAMESPACE" default:""`
	MetricsSubsystem string        `envconfig:"METRICS_SUBSYSTEM" default:""`
	MetricsPath      string        `envconfig:"METRICS_PATH" default:"/metrics"`
}

func main() {

	var log = logging.NewLogger()

	log.Println("volley-manager-public-api application is starting...")
	log.Printf("Version    : %s", version.Version)
	log.Printf("Commit     : %s", version.GitCommit)
	log.Printf("Build date : %s", version.BuildDate)
	log.Printf("OSarch     : %s", version.OsArch)

	var env EnvConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Errorf("Failed to process env var: %s", err)
		return
	}

	flag.Parse()
	err := logging.SetLogLevel(log, env.LogLevel)
	if err != nil {
		log.Errorf("Logging level %s do not seem to be right. Err = %v", env.LogLevel, err)
		return
	}

	if *setLogLevel != "" {
		logging.SetRemoteLogLevelAndExit(log, env.Port, *setLogLevel)
	}

	// errgroup will coordinate the many routines handling the API.
	cancellableCtx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(cancellableCtx)
	// signalChan will catch the shutdown signal and initiate a clean shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// The state where information are stored
	theState := newState(env.TeamsId)

	// The fetcher will poll the Volley Manager API at a given rate
	theFetcher, err := newFetcher(env.APIKey, theState)
	if err != nil {
		log.WithError(err).Error("Got an error while instantiating the fetcher")
	}

	// Loop
	g.Go(func() error {
		err := run(theFetcher, theState)
		if err != nil {
			return err
		}
		for range time.Tick(env.RefreshInterval) {
			err = run(theFetcher, theState)
			if err != nil {
				return err
			}
		}
		return nil
	})

	// The API will server the request with the data from the state
	theApi := newApi(theState)
	theApi.run(fmt.Sprintf("%s:%s", env.BindIP, env.Port), g)

	// Wait for any shutdown
	select {
	case <-signalChan:
		log.Info("Shutdown signal received, exiting...")
		cancel()
		break
	case <-ctx.Done():
		log.Info("Group context is done, exiting...")
		cancel()
		break
	}

	// if a non-clean shutdown was triggered, details are printed here
	err = ctx.Err()
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("Got an error from the error group context: %v", err)
	}
}

func run(f *fetcher, s *state) error {

	err := f.fetch()
	if err != nil {
		return err
	}

	teams := make(map[int]Team)
	gamesPerTeam := make(map[int][]Game)
	rankingPerTeam := make(map[int]GroupRankings)
	leagues := make(map[int]string)
	leaguePerTeam := make(map[int]League)
	groupPerTeam := make(map[int]Group)

	for _, game := range s.rawGames {
		leagues[game.League.LeagueId] = game.League.Translations.F

		if s.isManagedTeam(game.Teams.Away.TeamId) {
			teams[game.Teams.Away.TeamId] = game.Teams.Away
			l, ok := leaguePerTeam[game.Teams.Away.TeamId]
			if ok {
				if l.LeagueId != game.League.LeagueId {
					fmt.Printf("League mismatch for team %s: previous %s current %s\n", game.Teams.Away.Caption, l.Caption, game.League.Caption)
				}
			} else {
				leaguePerTeam[game.Teams.Away.TeamId] = game.League
			}
			g, ok := groupPerTeam[game.Teams.Away.TeamId]
			if ok {
				if g.GroupId != game.Group.GroupId {
					fmt.Printf("Group mismatch for team %s: previous %s current %s\n", game.Teams.Away.Caption, g.Caption, game.Group.Caption)
				}
			} else {
				groupPerTeam[game.Teams.Away.TeamId] = game.Group
			}
			t := gamesPerTeam[game.Teams.Away.TeamId]
			gamesPerTeam[game.Teams.Away.TeamId] = append(t, game)
		}

		if s.isManagedTeam(game.Teams.Home.TeamId) {
			teams[game.Teams.Home.TeamId] = game.Teams.Home
			l, ok := leaguePerTeam[game.Teams.Home.TeamId]
			if ok {
				if l.LeagueId != game.League.LeagueId {
					fmt.Printf("League mismatch for team %s: previous %s current %s\n", game.Teams.Home.Caption, l.Caption, game.League.Caption)
				}
			} else {
				leaguePerTeam[game.Teams.Home.TeamId] = game.League
			}
			gr, ok := groupPerTeam[game.Teams.Home.TeamId]
			if ok {
				if gr.GroupId != game.Group.GroupId {
					fmt.Printf("Group mismatch for team %s: previous %s current %s\n", game.Teams.Home.Caption, gr.Caption, game.Group.Caption)
				}
			} else {
				groupPerTeam[game.Teams.Home.TeamId] = game.Group
			}
			t := gamesPerTeam[game.Teams.Home.TeamId]
			gamesPerTeam[game.Teams.Home.TeamId] = append(t, game)
		}
	}

	for _, ranking := range s.rawRankings {
		for teamId, group := range groupPerTeam {
			if group.GroupId == ranking.GroupId {
				rankingPerTeam[teams[teamId].TeamId] = ranking
				for i, t := range ranking.Ranking {
					if t.TeamId == teamId {
						t.IsTeam = true
						ranking.Ranking[i] = t
					}
				}
				fmt.Printf("Team %s is in this %d, %d, %d, #ranking %d\n", teams[teamId].Caption, ranking.LeagueId, ranking.PhaseId, ranking.GroupId, len(ranking.Ranking))
				fmt.Printf("Ranking is %v\n", ranking)
			}
		}
	}

	s.lock.Lock()
	s.teams = teams
	s.gamesPerTeam = gamesPerTeam
	s.rankingPerTeam = rankingPerTeam
	s.leaguePerTeam = leaguePerTeam
	s.groupPerTeam = groupPerTeam
	s.lock.Unlock()

	return nil
}
