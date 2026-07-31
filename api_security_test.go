package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

type securityRoundTripper func(*http.Request) (*http.Response, error)

func (roundTrip securityRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	body := `[]`
	if request.URL.String() == gamesCollectionUri {
		body = `[
			{"gameId": 1, "playDate": "2099-01-01 12:00:00", "teams": {"home": {"teamId": 1, "caption": "Managed", "clubId": "managed"}, "away": {"teamId": 2, "caption": "Visitor", "clubId": "outside"}}},
			{"gameId": 2, "playDate": "2099-01-01 12:00:00", "teams": {"home": {"teamId": 3, "caption": "Unmanaged", "clubId": "outside"}, "away": {"teamId": 4, "caption": "Other", "clubId": "outside"}}}
		]`
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    request,
	}, nil
}

func securityFetcher() *fetcher {
	return &fetcher{
		httpClient: &http.Client{Transport: securityRoundTripper(func(request *http.Request) (*http.Response, error) {
			return securityRoundTripper(nil).RoundTrip(request)
		})},
	}
}

func TestRefreshPipeline_never_exposes_unmanaged_games(t *testing.T) {
	// Given a club-scoped deployment and a concurrent refresh loop
	gin.SetMode(gin.TestMode)
	s := newState("managed", nil, nil)
	a := newApi(s, nil)
	router := a.router()
	f := securityFetcher()

	ctx, cancel := context.WithCancel(context.Background())
	var refreshes sync.WaitGroup
	refreshes.Add(1)
	go func() {
		defer refreshes.Done()
		for range 200 {
			games, rankings, err := f.fetch(ctx)
			if err != nil {
				return
			}
			s.rebuildManagedGames(games, rankings)
		}
	}()

	// When readers hammer the global endpoints during refresh
	for range 200 {
		// Then only managed games are ever visible (run with -race)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/upcoming", nil))
		assert.NotContains(t, response.Body.String(), "Unmanaged")
		response = httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/past", nil))
		assert.NotContains(t, response.Body.String(), "Unmanaged")
	}
	cancel()
	refreshes.Wait()
}

func TestMCPHandler_is_stateless(t *testing.T) {
	// Given the production MCP handler
	s := newState("", nil, nil)
	handler := newMcpHandler(s, newGamePresenter(time.UTC, nil, s.isCup))
	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	// When an unauthenticated client initializes
	body := `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2025-11-25", "capabilities": {}, "clientInfo": {"name": "test", "version": "0"}}}`
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, testServer.URL, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(req)

	// Then no session is retained for the request
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("Mcp-Session-Id"))
}

func TestTeamEndpoints_reject_invalid_id_without_internal_details(t *testing.T) {
	// Given the production router
	gin.SetMode(gin.TestMode)
	s := newState("", nil, nil)
	router := newApi(s, nil).router()

	for _, path := range []string{"/upcoming/abc", "/past/abc", "/ranking/abc", "/ics/upcoming/abc"} {
		// When the team id is not numeric
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))

		// Then the client gets a generic error, not the raw input or parser internals
		assert.Equal(t, http.StatusBadRequest, response.Code, path)
		assert.NotContains(t, response.Body.String(), "abc", path)
		assert.NotContains(t, response.Body.String(), "strconv", path)
	}
}

func TestTeamICS_sanitizes_download_filename(t *testing.T) {
	// Given a team whose upstream caption carries header-hostile characters
	gin.SetMode(gin.TestMode)
	s := newState("", nil, nil)
	caption := "Evil\r\nInjected: \"Team\""
	s.teams = map[int]Team{7: {TeamId: 7, Caption: caption, ClubId: "club"}}
	s.gamesPerTeam = map[int][]Game{7: {}}
	router := newApi(s, nil).router()

	// When the calendar is downloaded
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ics/upcoming/7", nil))

	// Then the Content-Disposition header carries no control characters
	disposition := response.Header().Get("Content-Disposition")
	assert.NotContains(t, disposition, "\r")
	assert.NotContains(t, disposition, "\n")
	assert.Contains(t, disposition, "attachment; filename=")
}

func TestRouter_sets_security_headers(t *testing.T) {
	// Given the production router
	gin.SetMode(gin.TestMode)
	s := newState("", nil, nil)
	router := newApi(s, nil).router()

	// When any endpoint answers
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ping", nil))

	// Then baseline anti-MIME-sniffing and referrer headers are present
	assert.Equal(t, "nosniff", response.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", response.Header().Get("Referrer-Policy"))
}

func TestRouter_static_assets_require_revalidation(t *testing.T) {
	// Given the production router
	gin.SetMode(gin.TestMode)
	s := newState("", nil, nil)
	router := newApi(s, nil).router()

	// When a static asset is requested (present or not, the middleware runs first)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/static/js/ranking.js", nil))

	// Then the browser must revalidate it on every load
	assert.Equal(t, "no-cache", response.Header().Get("Cache-Control"))
}

func TestHTTPServer_has_timeouts(t *testing.T) {
	// Given the production HTTP server
	s := newState("", nil, nil)
	server := newApi(s, nil).newHTTPServer("127.0.0.1:0")

	// Then slow-client deadlines are configured
	assert.Positive(t, server.ReadHeaderTimeout)
	assert.Positive(t, server.ReadTimeout)
	assert.Positive(t, server.WriteTimeout)
	assert.Positive(t, server.IdleTimeout)
}

func TestRun_shuts_down_gracefully_when_context_is_canceled(t *testing.T) {
	// Given a running server on a free local port
	gin.SetMode(gin.TestMode)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	s := newState("", nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	g, groupCtx := errgroup.WithContext(ctx)
	newApi(s, nil).run(address, groupCtx, g)

	// When it serves a request and is then asked to stop
	var pong string
	require.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://%s/ping", address))
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		pong = string(body)
		return err == nil && resp.StatusCode == http.StatusOK
	}, 5*time.Second, 10*time.Millisecond)
	assert.Contains(t, pong, "pong")
	cancel()

	// Then the server shuts down without error
	require.NoError(t, g.Wait())
}
