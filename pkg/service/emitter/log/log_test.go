package log_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/tamamo/pkg/domain/model/event"
	logEmitter "github.com/secmon-lab/tamamo/pkg/service/emitter/log"
)

func TestEmitter(t *testing.T) {
	t.Run("emits event as structured log", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		e := logEmitter.New(logger)

		ev := &event.Event{
			Timestamp: time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
			NodeID:    "test-node",
			EventType: "http_request",
			SourceIP:  "10.0.0.1",
			Method:    "GET",
			Path:      "/login",
			Scenario:  "test-scenario",
		}

		err := e.Emit(context.Background(), ev)
		gt.NoError(t, err)

		var entry map[string]any
		gt.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
		gt.Equal(t, entry["msg"], "honeypot event")
		gt.Equal(t, entry["node_id"], "test-node")
		gt.Equal(t, entry["event_type"], "http_request")
		gt.Equal(t, entry["source_ip"], "10.0.0.1")
		gt.Equal(t, entry["method"], "GET")
		gt.Equal(t, entry["path"], "/login")
		gt.Equal(t, entry["scenario"], "test-scenario")
	})
}
