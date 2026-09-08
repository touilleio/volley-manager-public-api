package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readStaticContractFile(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	require.NoError(t, err, path)

	return string(contents)
}

func TestStaticDarkMode_contracts_design_and_pages(t *testing.T) {
	// Given the planned static design-system contract
	design := readStaticContractFile(t, "DESIGN.md")

	// Then the root design contract declares the eight numbered sections and brand colors
	for _, section := range []string{"## 1", "## 2", "## 3", "## 4", "## 5", "## 6", "## 7", "## 8"} {
		assert.Contains(t, design, section)
	}
	for _, color := range []string{"#417BA6", "#FDC500"} {
		assert.Contains(t, design, color)
	}

	pageContracts := map[string]string{
		"static/upcoming.html": "Prochains matchs",
		"static/past.html":     "Matchs joués",
		"static/ranking.html":  "Classement",
	}

	for path, title := range pageContracts {
		page := readStaticContractFile(t, path)

		// When the page is parsed as raw text
		assert.Contains(t, page, title)
		assert.True(t, strings.Index(page, `js/theme.js`) >= 0, path)
		assert.True(t, strings.Index(page, `js/theme.js`) < strings.Index(page, `css/main.css`), path)
		assert.Regexp(t, regexp.MustCompile(`(?s)<button[^>]*data-theme-toggle[^>]*type="button"[^>]*aria-label="[^"]+"[^>]*aria-pressed="(?:true|false)"[^>]*>.*?<svg\b.*?</svg>.*?</button>`), page)
		assert.Len(t, regexp.MustCompile(`data-theme-toggle`).FindAllString(page, -1), 1, path)
	}
}

func TestStaticDarkMode_contracts_theme_script(t *testing.T) {
	// Given the planned shared theme controller
	themeJS := readStaticContractFile(t, "static/js/theme.js")

	// Then the script stores the theme, follows system preference, and does not depend on jQuery
	assert.Contains(t, themeJS, "localStorage")
	assert.Contains(t, themeJS, "matchMedia")
	assert.Contains(t, themeJS, "prefers-color-scheme")
	assert.Contains(t, themeJS, "data-theme")
	assert.Contains(t, themeJS, "change")
	assert.NotContains(t, themeJS, "jQuery")
	assert.NotContains(t, themeJS, "window.$")
}

func TestStaticDarkMode_contracts_main_css(t *testing.T) {
	// Given the shared stylesheet contract
	mainCSS := readStaticContractFile(t, "static/css/main.css")

	// Then the stylesheet defines theme tokens, the dark theme selector, focus-visible toggle styling, and DataTables/cup selectors
	for _, token := range []string{"--color-background", "--color-surface", "--color-text", "--color-border", "--color-accent"} {
		assert.Contains(t, mainCSS, token)
	}
	assert.Contains(t, mainCSS, ":root")
	assert.Contains(t, mainCSS, `html[data-theme="dark"]`)
	assert.Contains(t, mainCSS, `button[data-theme-toggle]:focus-visible`)
	for _, selector := range []string{
		`div.dataTables_wrapper`,
		`div.dataTables_filter`,
		`div.dataTables_paginate`,
		`table.dataTable thead th`,
		`table.dataTable tbody tr.odd`,
		`table.dataTable tbody tr.even`,
		`table.dataTable tbody tr.child`,
		`table.dataTable tbody tr.selected`,
		`.league-cup-badge`,
	} {
		assert.Contains(t, mainCSS, selector)
	}
}
