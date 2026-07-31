package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// changeNotification is the formatting-independent payload published to SQS.
// Consumers own the rendering (Telegram message, email, ...).
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

type sqsClient interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type sqsPublisher struct {
	client   sqsClient
	queueURL string
}

func newSqsPublisher(ctx context.Context, queueURL string, region string) (*sqsPublisher, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	if cfg.Region == "" {
		return nil, errors.New("AWS region is not set: set AWS_REGION")
	}
	return &sqsPublisher{client: sqs.NewFromConfig(cfg), queueURL: queueURL}, nil
}

// publish sends one SQS message per poll batch. SQS is assumed available;
// callers log a failure and move on to the next poll.
func (p *sqsPublisher) publish(ctx context.Context, changes []GameChange) error {
	if len(changes) == 0 {
		return nil
	}
	payload, err := json.Marshal(newChangeNotification(changes, time.Now()))
	if err != nil {
		return fmt.Errorf("marshalling change notification: %w", err)
	}
	_, err = p.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &p.queueURL,
		MessageBody: aws.String(string(payload)),
	})
	if err != nil {
		return fmt.Errorf("publishing change notification to SQS: %w", err)
	}
	return nil
}
