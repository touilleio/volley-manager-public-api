package main

import (
	"context"
	"fmt"
	"io"
)

// notifier delivers game change notifications. Implementations must be safe
// to call from the fetch goroutine; delivery failures are logged by the
// caller and never abort the poll loop.
type notifier interface {
	notifyChanges(ctx context.Context, changes []GameChange) error
}

// logNotifier writes the notification message to out. It is the default when
// no Telegram credentials are configured, so moved matches stay visible.
type logNotifier struct {
	out io.Writer
}

func newLogNotifier(out io.Writer) logNotifier {
	return logNotifier{out: out}
}

func (l logNotifier) notifyChanges(_ context.Context, changes []GameChange) error {
	if len(changes) == 0 {
		return nil
	}
	_, err := fmt.Fprintln(l.out, formatChangesMessage(changes))
	return err
}
