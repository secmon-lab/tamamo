package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

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
					MinFailures:        2,
					SuccessProbability: 1.0, // always succeed after min_failures for deterministic testing
					FailureStatusCode:  401,
					FailureBody:        `{"success": false, "error": "Invalid credentials"}`,
					FailureHeaders:     map[string]string{"Content-Type": "application/json"},
					CredentialFields:   []string{"username", "password"},
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

	t.Run("auth succeeds after enough unique credentials", func(t *testing.T) {
		// min_failures=2, success_probability=1.0
		// First unique credential: must fail (failures=0 < 2)
		resp1, err := http.Post(ts.URL+"/api/auth/login", "application/json",
			strings.NewReader(`{"username":"admin","password":"wrong1"}`))
		gt.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()
		gt.Equal(t, resp1.StatusCode, 401)

		// Second unique credential: must fail (failures=1 < 2)
		resp2, err := http.Post(ts.URL+"/api/auth/login", "application/json",
			strings.NewReader(`{"username":"admin","password":"wrong2"}`))
		gt.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()
		gt.Equal(t, resp2.StatusCode, 401)

		// Third unique credential: succeed (failures=2 >= 2, probability=1.0)
		resp3, err := http.Post(ts.URL+"/api/auth/login", "application/json",
			strings.NewReader(`{"username":"admin","password":"correct"}`))
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

func newTestServer(routes ...scenario.Route) *httptest.Server {
	s := &scenario.Scenario{
		Meta:   scenario.Meta{Name: "test"},
		Routes: routes,
	}
	srv := honeypotHTTP.New(s, "node", []interfaces.Emitter{&mockEmitter{}})
	return httptest.NewServer(srv.Handler())
}

// authRoute with probability=1.0 for deterministic tests (always succeed after min_failures)
func deterministicAuthRoute() scenario.Route {
	return scenario.Route{
		Path:       "/auth/login",
		Method:     "POST",
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{"ok":true}`,
		Auth: &scenario.AuthStrategy{
			MinFailures:        1,
			SuccessProbability: 1.0,
			FailureStatusCode:  401,
			FailureBody:        `{"ok":false}`,
			FailureHeaders:     map[string]string{"Content-Type": "application/json"},
			CredentialFields:   []string{"username", "password"},
		},
	}
}

func TestAuthCredentialCounting(t *testing.T) {
	t.Run("same credential always returns same result", func(t *testing.T) {
		ts := newTestServer(deterministicAuthRoute())
		defer ts.Close()

		body := `{"username":"admin","password":"secret"}`

		// First attempt: fail (failures=0 < min_failures=1)
		resp1, err := http.Post(ts.URL+"/auth/login", "application/json", strings.NewReader(body))
		gt.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()
		gt.Equal(t, resp1.StatusCode, 401)

		// Same credential again: still fail (cached result)
		resp2, err := http.Post(ts.URL+"/auth/login", "application/json", strings.NewReader(body))
		gt.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()
		gt.Equal(t, resp2.StatusCode, 401)

		// Same credential a third time: still fail
		resp3, err := http.Post(ts.URL+"/auth/login", "application/json", strings.NewReader(body))
		gt.NoError(t, err)
		defer func() { _ = resp3.Body.Close() }()
		gt.Equal(t, resp3.StatusCode, 401)
	})

	t.Run("different credentials advance past min_failures", func(t *testing.T) {
		ts := newTestServer(deterministicAuthRoute())
		defer ts.Close()

		// First unique credential: fail (failures=0 < 1)
		resp1, err := http.Post(ts.URL+"/auth/login", "application/json",
			strings.NewReader(`{"username":"admin","password":"wrong"}`))
		gt.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()
		gt.Equal(t, resp1.StatusCode, 401)

		// Second unique credential: succeed (failures=1 >= 1, probability=1.0)
		resp2, err := http.Post(ts.URL+"/auth/login", "application/json",
			strings.NewReader(`{"username":"admin","password":"other"}`))
		gt.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()
		gt.Equal(t, resp2.StatusCode, 200)
	})

	t.Run("dummy fields do not affect credential counting", func(t *testing.T) {
		ts := newTestServer(deterministicAuthRoute())
		defer ts.Close()

		body1 := `{"username":"admin","password":"secret","csrf":"token1"}`
		body2 := `{"username":"admin","password":"secret","csrf":"token2"}`
		body3 := `{"username":"admin","password":"secret","extra":"data"}`

		resp1, err := http.Post(ts.URL+"/auth/login", "application/json", strings.NewReader(body1))
		gt.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()
		gt.Equal(t, resp1.StatusCode, 401)

		resp2, err := http.Post(ts.URL+"/auth/login", "application/json", strings.NewReader(body2))
		gt.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()
		gt.Equal(t, resp2.StatusCode, 401)

		resp3, err := http.Post(ts.URL+"/auth/login", "application/json", strings.NewReader(body3))
		gt.NoError(t, err)
		defer func() { _ = resp3.Body.Close() }()
		gt.Equal(t, resp3.StatusCode, 401)
	})

	t.Run("form-urlencoded submission works", func(t *testing.T) {
		ts := newTestServer(deterministicAuthRoute())
		defer ts.Close()

		// First credential via form: fail
		resp1, err := http.Post(ts.URL+"/auth/login", "application/x-www-form-urlencoded",
			strings.NewReader("username=admin&password=wrong"))
		gt.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()
		gt.Equal(t, resp1.StatusCode, 401)

		// Same credential: still fail (cached)
		resp2, err := http.Post(ts.URL+"/auth/login", "application/x-www-form-urlencoded",
			strings.NewReader("username=admin&password=wrong"))
		gt.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()
		gt.Equal(t, resp2.StatusCode, 401)

		// Different credential: succeed
		resp3, err := http.Post(ts.URL+"/auth/login", "application/x-www-form-urlencoded",
			strings.NewReader("username=admin&password=correct"))
		gt.NoError(t, err)
		defer func() { _ = resp3.Body.Close() }()
		gt.Equal(t, resp3.StatusCode, 200)
	})

	t.Run("form-urlencoded dummy fields do not affect counting", func(t *testing.T) {
		ts := newTestServer(deterministicAuthRoute())
		defer ts.Close()

		resp1, err := http.Post(ts.URL+"/auth/login", "application/x-www-form-urlencoded",
			strings.NewReader("username=admin&password=secret&csrf=aaa"))
		gt.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()
		gt.Equal(t, resp1.StatusCode, 401)

		resp2, err := http.Post(ts.URL+"/auth/login", "application/x-www-form-urlencoded",
			strings.NewReader("username=admin&password=secret&csrf=bbb"))
		gt.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()
		gt.Equal(t, resp2.StatusCode, 401)
	})

	t.Run("success result is also cached", func(t *testing.T) {
		ts := newTestServer(deterministicAuthRoute())
		defer ts.Close()

		// Fail with first credential
		resp1, err := http.Post(ts.URL+"/auth/login", "application/json",
			strings.NewReader(`{"username":"a","password":"1"}`))
		gt.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()
		gt.Equal(t, resp1.StatusCode, 401)

		// Succeed with second credential
		resp2, err := http.Post(ts.URL+"/auth/login", "application/json",
			strings.NewReader(`{"username":"b","password":"2"}`))
		gt.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()
		gt.Equal(t, resp2.StatusCode, 200)

		// Same successful credential: still succeed (cached)
		resp3, err := http.Post(ts.URL+"/auth/login", "application/json",
			strings.NewReader(`{"username":"b","password":"2"}`))
		gt.NoError(t, err)
		defer func() { _ = resp3.Body.Close() }()
		gt.Equal(t, resp3.StatusCode, 200)
	})

	t.Run("zero probability never succeeds after min_failures", func(t *testing.T) {
		route := deterministicAuthRoute()
		route.Auth.SuccessProbability = 0.0
		ts := newTestServer(route)
		defer ts.Close()

		// Send many different credentials, all should fail
		for i := range 5 {
			body := fmt.Sprintf(`{"username":"user%d","password":"pass%d"}`, i, i)
			resp, err := http.Post(ts.URL+"/auth/login", "application/json", strings.NewReader(body))
			gt.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			gt.Equal(t, resp.StatusCode, 401)
		}
	})

	t.Run("credential_fields unset falls back to full body hash", func(t *testing.T) {
		route := deterministicAuthRoute()
		route.Auth.CredentialFields = nil
		ts := newTestServer(route)
		defer ts.Close()

		body := `{"username":"admin","password":"secret"}`

		// Same body: always fail (only 1 unique body seen)
		resp1, err := http.Post(ts.URL+"/auth/login", "application/json", strings.NewReader(body))
		gt.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()
		gt.Equal(t, resp1.StatusCode, 401)

		resp2, err := http.Post(ts.URL+"/auth/login", "application/json", strings.NewReader(body))
		gt.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()
		gt.Equal(t, resp2.StatusCode, 401)

		// Different body: succeed (probability=1.0)
		resp3, err := http.Post(ts.URL+"/auth/login", "application/json",
			strings.NewReader(`{"username":"admin","password":"other"}`))
		gt.NoError(t, err)
		defer func() { _ = resp3.Body.Close() }()
		gt.Equal(t, resp3.StatusCode, 200)
	})
}

func TestHangRoute(t *testing.T) {
	t.Run("hang route does not respond until client cancels", func(t *testing.T) {
		ts := newTestServer(scenario.Route{
			Path:   "/dashboard",
			Method: "GET",
			Hang:   true,
		})
		defer ts.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", ts.URL+"/dashboard", nil)
		gt.NoError(t, err)

		_, err = http.DefaultClient.Do(req)
		gt.V(t, err).NotNil() // should fail due to context timeout
	})

	t.Run("hang route still emits event", func(t *testing.T) {
		mock := &mockEmitter{}
		s := &scenario.Scenario{
			Meta: scenario.Meta{Name: "hang-test"},
			Routes: []scenario.Route{
				{Path: "/dashboard", Method: "GET", Hang: true},
			},
		}
		srv := honeypotHTTP.New(s, "node", []interfaces.Emitter{mock})
		ts := httptest.NewServer(srv.Handler())
		defer ts.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", ts.URL+"/dashboard", nil)
		gt.NoError(t, err)
		_, _ = http.DefaultClient.Do(req)

		events := mock.getEvents()
		gt.True(t, len(events) > 0)
		gt.Equal(t, events[0].Path, "/dashboard")
	})
}

func TestFormBodyEventEmission(t *testing.T) {
	mock := &mockEmitter{}
	s := &scenario.Scenario{
		Meta: scenario.Meta{Name: "form-test"},
		Routes: []scenario.Route{
			{
				Path:       "/login",
				Method:     "POST",
				StatusCode: 200,
				Headers:    map[string]string{"Content-Type": "text/html"},
				Body:       "ok",
			},
		},
	}
	srv := honeypotHTTP.New(s, "node", []interfaces.Emitter{mock})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	t.Run("form-urlencoded body is emitted as map", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/login", "application/x-www-form-urlencoded",
			strings.NewReader("email=test%40example.com&password=secret"))
		gt.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		gt.Equal(t, resp.StatusCode, 200)

		events := mock.getEvents()
		gt.True(t, len(events) > 0)

		lastEvent := events[len(events)-1]
		bodyMap, ok := lastEvent.Body.(map[string]string)
		gt.V(t, ok).Equal(true)
		gt.Equal(t, bodyMap["email"], "test@example.com")
		gt.Equal(t, bodyMap["password"], "secret")
	})

	t.Run("json body is still emitted as parsed json", func(t *testing.T) {
		initialCount := len(mock.getEvents())

		resp, err := http.Post(ts.URL+"/login", "application/json",
			strings.NewReader(`{"key":"value"}`))
		gt.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		events := mock.getEvents()
		gt.True(t, len(events) > initialCount)

		lastEvent := events[len(events)-1]
		bodyMap, ok := lastEvent.Body.(map[string]any)
		gt.V(t, ok).Equal(true)
		gt.Equal(t, bodyMap["key"], "value")
	})

	t.Run("body is readable without Content-Length header", func(t *testing.T) {
		initialCount := len(mock.getEvents())

		req, err := http.NewRequest("POST", ts.URL+"/login", strings.NewReader("raw body data"))
		gt.NoError(t, err)
		req.ContentLength = -1
		req.Header.Set("Content-Type", "text/plain")

		resp, err := http.DefaultClient.Do(req)
		gt.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		_, _ = io.ReadAll(resp.Body)

		events := mock.getEvents()
		gt.True(t, len(events) > initialCount)

		lastEvent := events[len(events)-1]
		gt.V(t, lastEvent.Body).NotNil()
	})
}
