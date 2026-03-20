package config

import (
	"log/slog"
	"os"
	"strings"

	"github.com/secmon-lab/tamamo/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

// Logger holds logger configuration.
type Logger struct {
	Level string
}

// Flags returns CLI flags for logger configuration.
func (c *Logger) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "log-level",
			Usage:       "Log level (debug, info, warn, error)",
			Value:       "info",
			Sources:     cli.EnvVars("TAMAMO_LOG_LEVEL"),
			Destination: &c.Level,
		},
	}
}

// Configure creates a configured logger.
func (c *Logger) Configure() *slog.Logger {
	var level slog.Level
	switch strings.ToLower(c.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	return logging.New(level, os.Stderr)
}
