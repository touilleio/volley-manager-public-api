package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	telegramAPIBase       = "https://api.telegram.org"
	telegramMaxMessageLen = 4000 // Telegram hard limit is 4096 runes
)

// telegramNotifier sends one batched message per poll through a Telegram bot
// (https://core.telegram.org/bots/api#sendmessage), using plain HTTPS so no
// extra dependency is needed.
type telegramNotifier struct {
	botToken   string
	chatID     string
	apiBase    string // overridable for tests
	httpClient *http.Client
}

func newTelegramNotifier(botToken, chatID string) *telegramNotifier {
	return &telegramNotifier{
		botToken:   botToken,
		chatID:     chatID,
		apiBase:    telegramAPIBase,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *telegramNotifier) notifyChanges(ctx context.Context, changes []GameChange) error {
	if len(changes) == 0 {
		return nil
	}

	payload, err := json.Marshal(map[string]string{
		"chat_id": t.chatID,
		"text":    formatChangesMessage(changes),
	})
	if err != nil {
		return fmt.Errorf("marshalling telegram payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/bot%s/sendMessage", t.apiBase, t.botToken), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling telegram API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("telegram API returned %s: %s", resp.Status, string(body))
	}
	return nil
}

// formatChangesMessage renders one French message listing every changed game,
// e.g.:
//
//	🏐 Changement de match détecté
//
//	Gibloux Volley F1 vs Volley Fribourg
//	📅 Date : Samedi 20.09.2025 17h00 → Dimanche 21.09.2025 15h00
//	📍 Salle : Salle du Collège, Fribourg → Gymnase, Bulle
func formatChangesMessage(changes []GameChange) string {
	var b strings.Builder
	b.WriteString("🏐 Changement de match détecté")
	for _, gameChange := range changes {
		fmt.Fprintf(&b, "\n\n%s vs %s", gameChange.Game.Teams.Home.Caption, gameChange.Game.Teams.Away.Caption)
		for _, fc := range gameChange.Changes {
			b.WriteString("\n")
			switch fc.Field {
			case FieldPlayDate:
				fmt.Fprintf(&b, "📅 Date : %s → %s", formatPlayDate(fc.Old), formatPlayDate(fc.New))
			case FieldHall:
				fmt.Fprintf(&b, "📍 Salle : %s → %s", fc.Old, fc.New)
			case FieldHomeTeam:
				fmt.Fprintf(&b, "🏠 Domicile : %s → %s", fc.Old, fc.New)
			case FieldAwayTeam:
				fmt.Fprintf(&b, "✈️ Extérieur : %s → %s", fc.Old, fc.New)
			case FieldStatus:
				fmt.Fprintf(&b, "🔁 Statut : %s → %s", fc.Old, fc.New)
			}
		}
	}
	return truncateRunes(b.String(), telegramMaxMessageLen)
}

// formatPlayDate renders an API play date in the club's display format; on a
// parse error the raw value is returned unchanged.
func formatPlayDate(raw string) string {
	parsedTime, err := time.ParseInLocation(timeFormat, raw, clubLocation)
	if err != nil {
		return raw
	}
	return daysInFrench.Replace(parsedTime.Format(outputTimeFormat))
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "\n…"
}
