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
	theState := newState()

	// The fetcher will poll the Volley Manager API at a given rate
	theFetcher, err := newFetcher(theState)
	if err != nil {
		log.WithError(err).Error("Got an error while instantiating the fetcher")
	}
	theFetcher.run(env.RefreshInterval, g)

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
