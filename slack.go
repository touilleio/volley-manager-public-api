package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type slackNotifier struct {
	webhookURL string
	client     *http.Client
}

func newSlackNotifier(webhookURL string) *slackNotifier {
	return &slackNotifier{webhookURL: webhookURL, client: &http.Client{
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func (s *slackNotifier) send(ctx context.Context, text string) error {
	if s == nil || s.webhookURL == "" {
		return nil
	}
	payload, err := json.Marshal(struct {
		Text string `json:"text"`
	}{Text: "volley-manager-public-api: " + text})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(payload))
	if err != nil {
		return errors.New("invalid Slack webhook URL")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		// HTTP errors include the request URL, which contains the webhook secret.
		return errors.New("Slack webhook request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Slack webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func (s *slackNotifier) alert(ctx context.Context, text string) {
	if err := s.send(ctx, text); err != nil {
		slog.Warn("Could not send Slack failure notification", "err", err)
	}
}
