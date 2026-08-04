package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type changeNotification struct {
	Version    int                  `json:"version"`
	Type       string               `json:"type"`
	DetectedAt time.Time            `json:"detectedAt"`
	Games      []changedGamePayload `json:"games"`
}

type changedGamePayload struct {
	GameId   int           `json:"gameId"`
	PlayDate string        `json:"playDate"`
	HomeTeam string        `json:"homeTeam"`
	AwayTeam string        `json:"awayTeam"`
	League   string        `json:"league"`
	Hall     string        `json:"hall"`
	Changes  []FieldChange `json:"changes"`
}

type newGamesNotification struct {
	Version    int              `json:"version"`
	Type       string           `json:"type"`
	DetectedAt time.Time        `json:"detectedAt"`
	Games      []newGamePayload `json:"games"`
}

type newGamePayload struct {
	GameId   int    `json:"gameId"`
	PlayDate string `json:"playDate"`
	HomeTeam string `json:"homeTeam"`
	AwayTeam string `json:"awayTeam"`
	League   string `json:"league"`
	Hall     string `json:"hall"`
}

func newChangeNotification(changes []GameChange, detectedAt time.Time) changeNotification {
	notification := changeNotification{
		Version:    1,
		Type:       "volley.matches.changed",
		DetectedAt: detectedAt,
		Games:      make([]changedGamePayload, 0, len(changes)),
	}
	for _, gc := range changes {
		notification.Games = append(notification.Games, changedGamePayload{
			GameId:   gc.Game.GameId,
			PlayDate: gc.Game.PlayDate,
			HomeTeam: gc.Game.Teams.Home.Caption,
			AwayTeam: gc.Game.Teams.Away.Caption,
			League:   gc.Game.League.Caption,
			Hall:     hallSummary(gc.Game.Hall),
			Changes:  gc.Changes,
		})
	}
	return notification
}

func newNewGamesNotification(games []Game, detectedAt time.Time) newGamesNotification {
	notification := newGamesNotification{
		Version:    1,
		Type:       "volley.matches.new",
		DetectedAt: detectedAt,
		Games:      make([]newGamePayload, 0, len(games)),
	}
	for _, game := range games {
		notification.Games = append(notification.Games, newGamePayload{
			GameId:   game.GameId,
			PlayDate: game.PlayDate,
			HomeTeam: game.Teams.Home.Caption,
			AwayTeam: game.Teams.Away.Caption,
			League:   game.League.Caption,
			Hall:     hallSummary(game.Hall),
		})
	}
	return notification
}

type snsClient interface {
	Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

type snsPublisher struct {
	client   snsClient
	topicArn string
}

func newSnsPublisher(ctx context.Context, topicArn string, region string) (*snsPublisher, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	if cfg.Region == "" {
		return nil, errors.New("AWS region is not set: set AWS_REGION")
	}
	return &snsPublisher{client: sns.NewFromConfig(cfg), topicArn: topicArn}, nil
}

func (p *snsPublisher) publishChanges(ctx context.Context, changes []GameChange) error {
	if len(changes) == 0 {
		return nil
	}
	payload, err := json.Marshal(newChangeNotification(changes, time.Now()))
	if err != nil {
		return fmt.Errorf("marshalling change notification: %w", err)
	}
	return p.publish(ctx, payload, "change")
}

func (p *snsPublisher) publishNewGames(ctx context.Context, games []Game) error {
	if len(games) == 0 {
		return nil
	}
	payload, err := json.Marshal(newNewGamesNotification(games, time.Now()))
	if err != nil {
		return fmt.Errorf("marshalling new games notification: %w", err)
	}
	return p.publish(ctx, payload, "new games")
}

func (p *snsPublisher) publish(ctx context.Context, payload []byte, name string) error {
	message := string(payload)
	_, err := p.client.Publish(ctx, &sns.PublishInput{
		TopicArn: &p.topicArn,
		Message:  &message,
	})
	if err != nil {
		return fmt.Errorf("publishing %s notification to SNS: %w", name, err)
	}
	return nil
}
