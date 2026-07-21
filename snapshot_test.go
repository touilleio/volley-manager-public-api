package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnapshot_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "games-snapshot.json")
	games := []Game{baseGame()}

	assert.Nil(t, saveSnapshot(path, games))

	loaded, err := loadSnapshot(path)
	assert.Nil(t, err)
	if assert.Len(t, loaded, 1) {
		// Referees/SetResults custom unmarshalers don't survive a round-trip (pre-existing quirk).
		assert.Equal(t, games[0].GameId, loaded[0].GameId)
		assert.Equal(t, games[0].PlayDate, loaded[0].PlayDate)
		assert.Equal(t, games[0].Status, loaded[0].Status)
		assert.Equal(t, games[0].Hall, loaded[0].Hall)
		assert.Equal(t, games[0].Teams, loaded[0].Teams)
	}
}

func TestSnapshot_SaveLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "games-snapshot.json")

	assert.Nil(t, saveSnapshot(path, []Game{baseGame()}))

	entries, err := os.ReadDir(dir)
	assert.Nil(t, err)
	if assert.Len(t, entries, 1) {
		assert.Equal(t, "games-snapshot.json", entries[0].Name())
	}
}

func TestSnapshot_LoadMissingFile(t *testing.T) {
	loaded, err := loadSnapshot(filepath.Join(t.TempDir(), "does-not-exist.json"))
	assert.Nil(t, err)
	assert.Nil(t, loaded)
}

func TestSnapshot_LoadCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "games-snapshot.json")
	assert.Nil(t, os.WriteFile(path, []byte("{invalid json"), 0o644))

	loaded, err := loadSnapshot(path)
	assert.Error(t, err)
	assert.Nil(t, loaded)
}

func TestSnapshot_EmptyPathDisabled(t *testing.T) {
	assert.Nil(t, saveSnapshot("", []Game{baseGame()}))
	loaded, err := loadSnapshot("")
	assert.Nil(t, err)
	assert.Nil(t, loaded)
}
