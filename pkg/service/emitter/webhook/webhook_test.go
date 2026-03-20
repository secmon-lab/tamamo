package webhook_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/tamamo/pkg/domain/model/event"
	"github.com/secmon-lab/tamamo/pkg/service/emitter/webhook"
)

func TestEmitter(t *testing.T) {
	secret := "test-secret-key"

	t.Run("sends event with HMAC signature", func(t *testing.T) {
		var receivedBody []byte
		var receivedSig string
		var receivedContentType string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedContentType = r.Header.Get("Content-Type")
			receivedSig = r.Header.Get("X-Tamamo-Signature")
			body, err := io.ReadAll(r.Body)
			gt.NoError(t, err)
			receivedBody = body
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		e := webhook.New(server.URL, webhook.WithSecret(secret))
		ev := &event.Event{
			Timestamp: time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
			NodeID:    "test-node",
			EventType: "http_request",
			SourceIP:  "192.168.1.1",
			Method:    "POST",
			Path:      "/api/auth/login",
			Headers:   map[string]string{"User-Agent": "curl/8.0"},
			Body:      map[string]string{"username": "admin", "password": "test"},
			Scenario:  "test-scenario",
		}

		err := e.Emit(context.Background(), ev)
		gt.NoError(t, err)

		gt.Equal(t, receivedContentType, "application/json")

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(receivedBody)
		expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		gt.Equal(t, receivedSig, expectedSig)

		var receivedEvent event.Event
		gt.NoError(t, json.Unmarshal(receivedBody, &receivedEvent))
		gt.Equal(t, receivedEvent.NodeID, "test-node")
		gt.Equal(t, receivedEvent.SourceIP, "192.168.1.1")
		gt.Equal(t, receivedEvent.Path, "/api/auth/login")
	})

	t.Run("sends event without signature when no secret", func(t *testing.T) {
		var receivedSig string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedSig = r.Header.Get("X-Tamamo-Signature")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		e := webhook.New(server.URL)
		ev := &event.Event{
			Timestamp: time.Now(),
			EventType: "http_request",
		}

		err := e.Emit(context.Background(), ev)
		gt.NoError(t, err)
		gt.Equal(t, receivedSig, "")
	})

	t.Run("returns error on non-2xx status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		e := webhook.New(server.URL)
		ev := &event.Event{
			Timestamp: time.Now(),
			EventType: "http_request",
		}

		err := e.Emit(context.Background(), ev)
		gt.Error(t, err)
	})
}
