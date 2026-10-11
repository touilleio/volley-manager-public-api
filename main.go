package main

import (
	"context"
	"encoding/json"
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
	SnsTopicARN            string        `envconfig:"SNS_TOPIC_ARN" default:""`
	PublishNewGames        bool          `envconfig:"PUBLISH_NEW_GAMES" default:"false"`
	AwsRegion              string        `envconfig:"AWS_REGION" default:"eu-central-1"`
	StateSnapshotPath      string        `envconfig:"STATE_SNAPSHOT_PATH" default:"/data/games-snapshot.json"`
	SlackWebhookURL        string        `envconfig:"SLACK_WEBHOOK_URL" default:""`
}

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("Application failed", "err", err)
		newSlackNotifier(os.Getenv("SLACK_WEBHOOK_URL")).alert(context.Background(), "Application failed: "+err.Error())
		os.Exit(1)
	}
}

func run() error {

	var logLevel = new(slog.LevelVar)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	slog.Info("volley-manager-public-api application is starting...",
		"version", buildinfo.Version,
		"commit", buildinfo.GitCommit,
		"buildDate", buildinfo.BuildDate,
		"osArch", buildinfo.OSArch,
	)

	var env EnvConfig
	if err := envconfig.Process("", &env); err != nil {
		return fmt.Errorf("processing environment: %w", err)
	}
	if env.RefreshInterval <= 0 {
		return errors.New("REFRESH_INTERVAL must be greater than zero")
	}

	flag.Parse()
	logLevel.Set(parseLogLevel(env.LogLevel))

	if *setLogLevel != "" {
		logLevel.Set(parseLogLevel(*setLogLevel))
		slog.Info("Log level changed", "level", *setLogLevel)
	}

	// errgroup will coordinate the many routines handling the API.
	cancellableCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	g, ctx := errgroup.WithContext(cancellableCtx)

	// The state where information are stored
	theState := newState(env.ClubID, env.ExcludedTeamIDs, env.CupLeagueCategoryIDs)

	var publisher *snsPublisher
	if env.SnsTopicARN != "" {
		p, err := newSnsPublisher(cancellableCtx, env.SnsTopicARN, env.AwsRegion)
		if err != nil {
			return fmt.Errorf("instantiating SNS publisher: %w", err)
		}
		publisher = p
	} else {
		slog.Info("SNS_TOPIC_ARN is not set, match notifications are disabled")
	}

	// The change detector is primed with the snapshot of the previous run, so
	// that matches moved while the application was down are caught on the
	// first poll after a restart. A missing snapshot establishes a fresh baseline;
	// a corrupt snapshot stops startup so pending notifications are not discarded.
	snapshot, err := loadPollSnapshot(env.StateSnapshotPath)
	if err != nil {
		return fmt.Errorf("loading games snapshot: %w", err)
	}
	detector := newChangeDetector(snapshot.Games)

	// The fetcher will poll the Volley Manager API at a given rate
	theFetcher := newFetcher(env.APIKey)
	poller := poller{
		fetcher:         theFetcher,
		state:           theState,
		detector:        detector,
		publisher:       publisher,
		publishNewGames: env.PublishNewGames,
		snapshotPath:    env.StateSnapshotPath,
		pending:         snapshot.Pending,
		slack:           newSlackNotifier(env.SlackWebhookURL),
	}

	// First fetch must complete
	if err := poller.run(cancellableCtx); err != nil {
		return fmt.Errorf("initial poll: %w", err)
	}

	// Fetch loop
	g.Go(func() error {
		ticker := time.NewTicker(env.RefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				if err := poller.run(ctx); err != nil {
					slog.Warn("Got an error while fetching the data. Keeping the old version instead of terminating here.", "err", err)
				}
			}
		}
	})

	// The API will serve requests with the data from the state
	theApi := newApi(theState, env.TeamCaptionReplacement)
	theApi.run(fmt.Sprintf("%s:%s", env.BindIP, env.Port), ctx, g)

	// Wait for any shutdown
	<-ctx.Done()
	slog.Info("Shutdown requested, exiting...")
	cancel()

	// if a non-clean shutdown was triggered, details are printed here
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

type poller struct {
	fetcher         *fetcher
	state           *state
	detector        *changeDetector
	publisher       *snsPublisher
	publishNewGames bool
	snapshotPath    string
	pending         []json.RawMessage
	slack           *slackNotifier
	fetchFailures   int
	fetchAlerted    bool
}

func (p *poller) run(ctx context.Context) error {
	games, rankings, err := p.fetcher.fetch(ctx)
	if err != nil {
		if ctx.Err() == nil {
			p.fetchFailures++
			if p.fetchFailures > 10 && !p.fetchAlerted {
				if alertErr := p.slack.send(ctx, fmt.Sprintf("Upstream fetch failed %d consecutive times. Last error: %s", p.fetchFailures, err)); alertErr != nil {
					slog.Warn("Could not send Slack fetch failure notification", "err", alertErr)
				} else {
					p.fetchAlerted = true
				}
			}
			p.deliverPending(ctx)
		}
		return err
	}
	p.fetchFailures = 0
	p.fetchAlerted = false

	allGames := p.state.rebuildManagedGames(games, rankings)

	slog.Info("Pulled", "games", len(p.state.rawGames), "teams", len(p.state.teams))

	previousDetector := *p.detector
	now := time.Now()
	diff := p.detector.diff(allGames, now)
	pending := append([]json.RawMessage(nil), p.pending...)
	if p.publisher != nil {
		// One event per game avoids replaying an entire batch after partial delivery.
		for _, change := range diff.Changes {
			payload, err := pendingNotification(newChangeNotification([]GameChange{change}, now))
			if err != nil {
				*p.detector = previousDetector
				return err
			}
			pending = append(pending, payload)
		}
		if p.publishNewGames {
			for _, game := range diff.NewGames {
				payload, err := pendingNotification(newNewGamesNotification([]Game{game}, now))
				if err != nil {
					*p.detector = previousDetector
					return err
				}
				pending = append(pending, payload)
			}
		}
	}
	// Commit the baseline and pending events together before attempting SNS.
	if err := savePollSnapshot(p.snapshotPath, pollSnapshot{Games: allGames, Pending: pending}); err != nil {
		*p.detector = previousDetector
		p.slack.alert(ctx, "Could not save games and pending notifications: "+err.Error())
		return err
	}
	p.pending = pending
	p.deliverPending(ctx)

	return nil
}

func (p *poller) deliverPending(ctx context.Context) {
	if p.publisher == nil {
		return
	}
	for len(p.pending) > 0 {
		if err := p.publisher.publish(ctx, p.pending[0], "pending match"); err != nil {
			slog.Warn("Error publishing notification; retained for retry", "err", err)
			if ctx.Err() == nil {
				p.slack.alert(ctx, "SNS publish failed; notification retained for retry: "+err.Error())
			}
			return
		}
		remaining := p.pending[1:]
		if err := savePollSnapshot(p.snapshotPath, pollSnapshot{Games: p.detector.previous, Pending: remaining}); err != nil {
			slog.Warn("Could not acknowledge published notification; retained for retry", "err", err)
			p.slack.alert(ctx, "Could not acknowledge published notification: "+err.Error())
			return
		}
		p.pending = remaining
	}
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
