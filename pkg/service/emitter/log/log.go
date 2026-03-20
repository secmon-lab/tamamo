package log

import (
	"context"
	"log/slog"

	"github.com/secmon-lab/tamamo/pkg/domain/model/event"
)

// Emitter outputs events as structured log entries via slog.
type Emitter struct {
	logger *slog.Logger
}

// New creates a log Emitter with the given logger.
func New(logger *slog.Logger) *Emitter {
	return &Emitter{logger: logger}
}

// Emit writes the event as a structured log entry.
func (e *Emitter) Emit(_ context.Context, ev *event.Event) error {
	e.logger.Info("honeypot event",
		slog.String("event_type", ev.EventType),
		slog.String("node_id", ev.NodeID),
		slog.String("source_ip", ev.SourceIP),
		slog.String("method", ev.Method),
		slog.String("path", ev.Path),
		slog.String("scenario", ev.Scenario),
		slog.Time("timestamp", ev.Timestamp),
	)
	return nil
}
