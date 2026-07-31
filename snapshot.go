package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// loadSnapshot reads the games snapshot written by the previous run. A
// missing file or an empty path is not an error: it returns nil games, which
// leaves the change detector unprimed (fresh baseline).
func loadSnapshot(path string) ([]Game, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading snapshot %s: %w", path, err)
	}
	var games []Game
	if err := json.Unmarshal(data, &games); err != nil {
		return nil, fmt.Errorf("decoding snapshot %s: %w", path, err)
	}
	return games, nil
}

// saveSnapshot atomically writes the current games so a restart can diff the
// next poll against them. An empty path disables persistence.
func saveSnapshot(path string, games []Game) error {
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
	if err := json.NewEncoder(tmp).Encode(games); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("encoding snapshot: %w", err)
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
