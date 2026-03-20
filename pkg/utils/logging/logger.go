package logging

import (
	"io"
	"log/slog"

	"github.com/m-mizutani/clog"
)

// New creates a new structured logger with the given level and writer.
func New(level slog.Level, w io.Writer) *slog.Logger {
	handler := clog.New(
		clog.WithWriter(w),
		clog.WithLevel(level),
	)
	return slog.New(handler)
}
