package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/m-mizutani/gt"
	honeypotHTTP "github.com/secmon-lab/tamamo/pkg/controller/http"
	"github.com/secmon-lab/tamamo/pkg/domain/interfaces"
	"github.com/secmon-lab/tamamo/pkg/domain/model/event"
	"github.com/secmon-lab/tamamo/pkg/domain/model/scenario"
)

// mockEmitter captures emitted events for testing.
type mockEmitter struct {
	mu     sync.Mutex
	events []*event.Event
}

func (m *mockEmitter) Emit(_ context.Context, ev *event.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, ev)
	return nil
}

func (m *mockEmitter) getEvents() []*event.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*event.Event{}, m.events...)
}

func testScenario() *scenario.Scenario {
	return &scenario.Scenario{
		Meta: scenario.Meta{
			Name:            "Test Portal",
			ServerSignature: "nginx/1.24.0",
			Headers: map[string]string{
				"X-Powered-By": "Express",
			},
		},
		Pages: []scenario.Page{
			{Path: "/login", HTMLFile: "pages/login.html", ContentType: "text/html"},
		},
		Routes: []scenario.Route{
			{
				Path:       "/",
				Method:     "GET",
				StatusCode: 302,
				Headers:    map[string]string{"Location": "/login"},
				Body:       "",
			},
			{
				Path:       "/login",
				Method:     "GET",
				StatusCode: 200,
				Headers:    map[string]string{"Content-Type": "text/html"},
				Body:       "<html><body>Login</body></html>",
			},
			{
				Path:       "/api/auth/login",
				Method:     "POST",
				StatusCode: 200,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"success": true, "redirect": "/dashboard"}`,
				Auth: &scenario.AuthStrategy{
					FailuresBeforeSuccess: 2,
					FailureStatusCode:     401,
					FailureBody:           `{"success": false, "error": "Invalid credentials"}`,
					FailureHeaders:        map[string]string{"Content-Type": "application/json"},
				},
			},
			{
				Path:       "/api/health",
				Method:     "GET",
				StatusCode: 200,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"status": "ok"}`,
			},
		},
	}
}

func TestHTTPServer(t *testing.T) {
	mock := &mockEmitter{}
	srv := honeypotHTTP.New(testScenario(), "test-node", []interfaces.Emitter{mock})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	t.Run("root redirects to login", func(t *testing.T) {
		client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
		resp, err := client.Get(ts.URL + "/")
		gt.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		gt.Equal(t, resp.StatusCode, 302)
		gt.Equal(t, resp.Header.Get("Location"), "/login")
	})

	t.Run("login page returns HTML", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/login")
		gt.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		gt.Equal(t, resp.StatusCode, 200)
		gt.Equal(t, resp.Header.Get("Content-Type"), "text/html")
	})

	t.Run("server signature header is set", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/login")
		gt.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		gt.Equal(t, resp.Header.Get("Server"), "nginx/1.24.0")
		gt.Equal(t, resp.Header.Get("X-Powered-By"), "Express")
	})

	t.Run("login API fails then succeeds with auth strategy", func(t *testing.T) {
		body := `{"username":"admin","password":"P@ssw0rd"}`

		// First attempt: should fail
		resp1, err := http.Post(ts.URL+"/api/auth/login", "application/json", strings.NewReader(body))
		gt.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()
		gt.Equal(t, resp1.StatusCode, 401)
		var fail1 map[string]any
		gt.NoError(t, json.NewDecoder(resp1.Body).Decode(&fail1))
		gt.Equal(t, fail1["success"], false)

		// Second attempt: should also fail
		resp2, err := http.Post(ts.URL+"/api/auth/login", "application/json", strings.NewReader(body))
		gt.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()
		gt.Equal(t, resp2.StatusCode, 401)

		// Third attempt: should succeed
		resp3, err := http.Post(ts.URL+"/api/auth/login", "application/json", strings.NewReader(body))
		gt.NoError(t, err)
		defer func() { _ = resp3.Body.Close() }()
		gt.Equal(t, resp3.StatusCode, 200)
		var success map[string]any
		gt.NoError(t, json.NewDecoder(resp3.Body).Decode(&success))
		gt.Equal(t, success["success"], true)
	})

	t.Run("health endpoint returns ok", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/health")
		gt.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		gt.Equal(t, resp.StatusCode, 200)

		var result map[string]string
		gt.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		gt.Equal(t, result["status"], "ok")
	})

	t.Run("events are emitted for requests", func(t *testing.T) {
		// Clear previous events by getting initial count
		initialCount := len(mock.getEvents())

		_, err := http.Get(ts.URL + "/login")
		gt.NoError(t, err)

		events := mock.getEvents()
		gt.True(t, len(events) > initialCount)

		lastEvent := events[len(events)-1]
		gt.Equal(t, lastEvent.EventType, "http_request")
		gt.Equal(t, lastEvent.NodeID, "test-node")
		gt.Equal(t, lastEvent.Method, "GET")
		gt.Equal(t, lastEvent.Path, "/login")
		gt.Equal(t, lastEvent.Scenario, "Test Portal")
	})

	t.Run("unknown routes return 404", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/nonexistent")
		gt.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		gt.Equal(t, resp.StatusCode, 404)
	})
}
