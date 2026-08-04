package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessLogger_emitsStructuredRecord_whenRequestCompletes(t *testing.T) {
	// Given
	logger, logs := testJSONLogger()
	router := gin.New()
	router.Use(accessLogger(logger))
	router.GET("/resource", func(c *gin.Context) {
		c.String(http.StatusCreated, "created")
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/resource?token=secret", nil)
	request.RemoteAddr = "198.51.100.7:1234"

	// When
	router.ServeHTTP(recorder, request)

	// Then
	records := parseJSONLogs(t, logs)
	require.Len(t, records, 1)
	assert.Equal(t, "INFO", records[0]["level"])
	assert.Equal(t, "http.request", records[0]["msg"])
	assert.Equal(t, http.MethodGet, records[0]["method"])
	assert.Equal(t, "/resource", records[0]["path"])
	assert.EqualValues(t, http.StatusCreated, records[0]["status"])
	assert.EqualValues(t, len("created"), records[0]["response_bytes"])
	assert.NotEmpty(t, records[0]["duration"])
	assert.Equal(t, "198.51.100.7", records[0]["client_ip"])
	assert.NotContains(t, records[0], "query")
	assert.NotContains(t, records[0], "headers")
}

func TestRecoveryLogger_emitsStructuredErrorAnd500_whenHandlerPanics(t *testing.T) {
	// Given
	logger, logs := testJSONLogger()
	router := gin.New()
	router.Use(accessLogger(logger), recoveryLogger(logger))
	router.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/panic?token=secret", nil)

	// When
	router.ServeHTTP(recorder, request)

	// Then
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	records := parseJSONLogs(t, logs)
	require.Len(t, records, 2)
	assert.Equal(t, "ERROR", records[0]["level"])
	assert.Equal(t, "http.panic", records[0]["msg"])
	assert.Equal(t, "boom", records[0]["panic"])
	assert.Contains(t, records[0]["stack"], "TestRecoveryLogger")
	assert.Equal(t, "ERROR", records[1]["level"])
	assert.Equal(t, "http.request", records[1]["msg"])
	assert.EqualValues(t, http.StatusInternalServerError, records[1]["status"])
	assert.Equal(t, "/panic", records[1]["path"])
	assert.NotContains(t, records[0], "query")
	assert.NotContains(t, records[1], "query")
}

func TestAccessLogger_choosesLevelFromStatus_whenRequestCompletes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		level      string
	}{
		{name: "successful responses are info", statusCode: http.StatusNoContent, level: "INFO"},
		{name: "client errors are warn", statusCode: http.StatusNotFound, level: "WARN"},
		{name: "server errors are error", statusCode: http.StatusBadGateway, level: "ERROR"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			logger, logs := testJSONLogger()
			router := gin.New()
			router.Use(accessLogger(logger))
			router.GET("/status", func(c *gin.Context) {
				c.Status(test.statusCode)
			})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/status", nil)

			// When
			router.ServeHTTP(recorder, request)

			// Then
			records := parseJSONLogs(t, logs)
			require.Len(t, records, 1)
			assert.Equal(t, test.level, records[0]["level"])
			assert.EqualValues(t, test.statusCode, records[0]["status"])
		})
	}
}

func testJSONLogger() (*slog.Logger, *bytes.Buffer) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return logger, &logs
}

func parseJSONLogs(t *testing.T, logs *bytes.Buffer) []map[string]any {
	t.Helper()
	records := make([]map[string]any, 0)
	scanner := bufio.NewScanner(strings.NewReader(logs.String()))
	for scanner.Scan() {
		var record map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &record))
		records = append(records, record)
	}
	require.NoError(t, scanner.Err())
	return records
}
