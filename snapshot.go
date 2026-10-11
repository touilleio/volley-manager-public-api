package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type pollSnapshot struct {
	Version int               `json:"version"`
	Games   []Game            `json:"games"`
	Pending []json.RawMessage `json:"pending,omitempty"`
}

// loadSnapshot reads the games snapshot written by the previous run. A
// missing file or an empty path is not an error: it returns nil games, which
// leaves the change detector unprimed (fresh baseline).
func loadSnapshot(path string) ([]Game, error) {
	snapshot, err := loadPollSnapshot(path)
	return snapshot.Games, err
}

func loadPollSnapshot(path string) (pollSnapshot, error) {
	var snapshot pollSnapshot
	if path == "" {
		return snapshot, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return snapshot, nil
	}
	if err != nil {
		return snapshot, fmt.Errorf("reading snapshot %s: %w", path, err)
	}
	// Older releases stored only the games array.
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("[")) || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		err = json.Unmarshal(data, &snapshot.Games)
	} else {
		err = json.Unmarshal(data, &snapshot)
		if err == nil && snapshot.Version != 1 {
			err = fmt.Errorf("unsupported snapshot version %d", snapshot.Version)
		}
	}
	if err != nil {
		return pollSnapshot{}, fmt.Errorf("decoding snapshot %s: %w", path, err)
	}
	return snapshot, nil
}

// saveSnapshot atomically writes the current games so a restart can diff the
// next poll against them. An empty path disables persistence.
func saveSnapshot(path string, games []Game) error {
	return savePollSnapshot(path, pollSnapshot{Games: games})
}

func savePollSnapshot(path string, snapshot pollSnapshot) error {
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating snapshot dir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".games-snapshot-*")
	if err != nil {
		return fmt.Errorf("creating temp snapshot in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	snapshot.Version = 1
	if err := json.NewEncoder(tmp).Encode(snapshot); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("encoding snapshot: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("syncing snapshot: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("closing temp snapshot: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("renaming snapshot to %s: %w", path, err)
	}
	return nil
}
