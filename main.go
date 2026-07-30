package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/touilleio/volley-manager-public-api/internal/buildinfo"
	"golang.org/x/sync/errgroup"
)

var (
	setLogLevel = flag.String("set-log-level", "", "Change log level. Possible values are trace,debug,info,warn,error,fatal,panic")
)

type EnvConfig struct {
	APIKey                 string        `envconfig:"API_KEY"`
	RefreshInterval        time.Duration `envconfig:"REFRESH_INTERVAL" default:"1h"`
	ClubID                 string        `envconfig:"CLUB_ID" default:""`
	ExcludedTeamIDs        []int         `envconfig:"EXCLUDED_TEAMS_ID" default:""`
	CupLeagueCategoryIDs   []int         `envconfig:"CUP_LEAGUE_CATEGORY_IDS" default:"4"`
	TeamCaptionReplacement []string      `envconfig:"TEAM_CAPTION_REPLACEMENT" default:""`
	BindIP                 string        `envconfig:"BIND_IP" default:"0.0.0.0"`
	Port                   string        `envconfig:"PORT" default:"8080"`
	LogLevel               string        `envconfig:"LOG_LEVEL" default:"info"`
	MetricsNamespace       string        `envconfig:"METRICS_NAMESPACE" default:""`
	MetricsSubsystem       string        `envconfig:"METRICS_SUBSYSTEM" default:""`
	MetricsPath            string        `envconfig:"METRICS_PATH" default:"/metrics"`
	SqsQueueURL            string        `envconfig:"SQS_QUEUE_URL" default:""`
	AwsRegion              string        `envconfig:"AWS_REGION" default:"eu-central-1"`
	StateSnapshotPath      string        `envconfig:"STATE_SNAPSHOT_PATH" default:"/data/games-snapshot.json"`
}

func main() {

	var logLevel = new(slog.LevelVar)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	slog.Info("volley-manager-public-api application is starting...",
		"version", buildinfo.Version,
		"commit", buildinfo.GitCommit,
		"buildDate", buildinfo.BuildDate,
		"osArch", buildinfo.OSArch,
	)

	var env EnvConfig
	if err := envconfig.Process("", &env); err != nil {
		slog.Error("Failed to process env var", "err", err)
		return
	}

	flag.Parse()
	logLevel.Set(parseLogLevel(env.LogLevel))

	if *setLogLevel != "" {
		logLevel.Set(parseLogLevel(*setLogLevel))
		slog.Info("Log level changed", "level", *setLogLevel)
	}

	// errgroup will coordinate the many routines handling the API.
	cancellableCtx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(cancellableCtx)
	// signalChan will catch the shutdown signal and initiate a clean shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// The state where information are stored
	theState := newState(env.ClubID, env.ExcludedTeamIDs, env.CupLeagueCategoryIDs)

	// Change notifications are published to SQS; without a queue URL they are disabled.
	var publisher *sqsPublisher
	if env.SqsQueueURL != "" {
		p, err := newSqsPublisher(cancellableCtx, env.SqsQueueURL, env.AwsRegion)
		if err != nil {
			slog.Error("Got an error while instantiating the SQS publisher", "err", err)
			return
		}
		publisher = p
	} else {
		slog.Info("SQS_QUEUE_URL is not set, match change notifications are disabled")
	}

	// The change detector is primed with the snapshot of the previous run, so
	// that matches moved while the application was down are caught on the
	// first poll after a restart. A missing or corrupt snapshot simply means
	// the first poll establishes a fresh baseline without notifying.
	previousGames, err := loadSnapshot(env.StateSnapshotPath)
	if err != nil {
		slog.Warn("Could not load games snapshot, starting with a fresh baseline", "path", env.StateSnapshotPath, "err", err)
	}
	detector := newChangeDetector(previousGames)

	// The fetcher will poll the Volley Manager API at a given rate
	theFetcher, err := newFetcher(env.APIKey, theState)
	if err != nil {
		slog.Error("Got an error while instantiating the fetcher", "err", err)
		return
	}

	// First fetch must complete
	err = run(cancellableCtx, theFetcher, theState, detector, publisher, env.StateSnapshotPath)
	if err != nil {
		slog.Error("Got an error while fetching the data for the first time", "err", err)
		return
	}

	// Fetch loop
	g.Go(func() error {
		for range time.Tick(env.RefreshInterval) {
			err = run(ctx, theFetcher, theState, detector, publisher, env.StateSnapshotPath)
			if err != nil {
				slog.Warn("Got an error while fetching the data. Keeping the old version instead of terminating here.", "err", err)
			}
		}
		return nil
	})

	// The API will server the request with the data from the state
	theApi := newApi(theState, env.TeamCaptionReplacement)
	theApi.run(fmt.Sprintf("%s:%s", env.BindIP, env.Port), g)

	// Wait for any shutdown
	select {
	case <-signalChan:
		slog.Info("Shutdown signal received, exiting...")
		cancel()
		break
	case <-ctx.Done():
		slog.Info("Group context is done, exiting...")
		cancel()
		break
	}

	// if a non-clean shutdown was triggered, details are printed here
	err = ctx.Err()
	if err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("Got an error from the error group context", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, f *fetcher, s *state, detector *changeDetector, publisher *sqsPublisher, snapshotPath string) error {

	err := f.fetch(ctx)
	if err != nil {
		return err
	}

	allGames := s.rebuildManagedGames()

	slog.Info("Pulled", "games", len(s.rawGames), "teams", len(s.teams))

	// Publication and snapshot failures never fail the poll; fresh data is already live.
	if changes := detector.diff(allGames, time.Now()); len(changes) > 0 {
		slog.Info("Detected changed game(s)", "count", len(changes))
		if publisher != nil {
			if err := publisher.publish(ctx, changes); err != nil {
				slog.Warn("Error publishing change notification", "err", err)
			}
		}
	}
	if err := saveSnapshot(snapshotPath, allGames); err != nil {
		slog.Warn("Error saving games snapshot", "err", err)
	}

	return nil
}

// parseLogLevel also accepts the previous logger's level names (trace/fatal/panic).
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "trace", "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error", "fatal", "panic":
		return slog.LevelError
	default:
		slog.Warn("Unknown log level, falling back to info", "level", level)
		return slog.LevelInfo
	}
}
