package config

import (
	"log/slog"
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
	"github.com/urfave/cli/v3"
)

// Prompt holds prompt customization configuration.
type Prompt struct {
	SiteType    string
	Style       string
	Taste       string
	Layout      string
	Lang        string
	ExtraPrompt string
	PromptFile  string
}

// Flags returns CLI flags for prompt configuration.
func (c *Prompt) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "site-type",
			Usage:       "Site type (e.g., \"IT asset management portal\")",
			Sources:     cli.EnvVars("TAMAMO_SITE_TYPE"),
			Destination: &c.SiteType,
		},
		&cli.StringFlag{
			Name:        "site-style",
			Usage:       "Visual style (e.g., \"corporate-minimal\", \"dark-tech\")",
			Sources:     cli.EnvVars("TAMAMO_SITE_STYLE"),
			Destination: &c.Style,
		},
		&cli.StringFlag{
			Name:        "site-taste",
			Usage:       "Taste/atmosphere (e.g., \"Large enterprise\", \"Startup\")",
			Sources:     cli.EnvVars("TAMAMO_SITE_TASTE"),
			Destination: &c.Taste,
		},
		&cli.StringFlag{
			Name:        "site-layout",
			Usage:       "Page layout pattern (e.g., \"split-screen\", \"terminal-cli\")",
			Sources:     cli.EnvVars("TAMAMO_SITE_LAYOUT"),
			Destination: &c.Layout,
		},
		&cli.StringFlag{
			Name:        "site-lang",
			Usage:       "Display language (e.g., \"English\", \"Japanese\")",
			Sources:     cli.EnvVars("TAMAMO_SITE_LANG"),
			Destination: &c.Lang,
		},
		&cli.StringFlag{
			Name:        "extra-prompt",
			Usage:       "Additional free-form prompt text",
			Sources:     cli.EnvVars("TAMAMO_EXTRA_PROMPT"),
			Destination: &c.ExtraPrompt,
		},
		&cli.StringFlag{
			Name:        "prompt-file",
			Usage:       "Path to a file containing additional prompt text",
			Destination: &c.PromptFile,
		},
	}
}

// LogConfig logs the prompt configuration (only set values).
func (c *Prompt) LogConfig(logger *slog.Logger) {
	attrs := []any{}
	if c.SiteType != "" {
		attrs = append(attrs, slog.String("site_type", c.SiteType))
	}
	if c.Style != "" {
		attrs = append(attrs, slog.String("site_style", c.Style))
	}
	if c.Taste != "" {
		attrs = append(attrs, slog.String("site_taste", c.Taste))
	}
	if c.Layout != "" {
		attrs = append(attrs, slog.String("site_layout", c.Layout))
	}
	if c.Lang != "" {
		attrs = append(attrs, slog.String("site_lang", c.Lang))
	}
	if c.PromptFile != "" {
		attrs = append(attrs, slog.String("prompt_file", c.PromptFile))
	}
	if len(attrs) > 0 {
		logger.Info("prompt configuration", attrs...)
	}
}

// ResolveExtraPrompt returns the extra prompt, reading from file if needed.
func (c *Prompt) ResolveExtraPrompt() (string, error) {
	if c.PromptFile != "" {
		data, err := os.ReadFile(c.PromptFile)
		if err != nil {
			return "", goerr.Wrap(err, "failed to read prompt file",
				goerr.V("path", c.PromptFile),
				goerr.T(errutil.TagNotFound),
			)
		}
		if c.ExtraPrompt != "" {
			return c.ExtraPrompt + "\n" + string(data), nil
		}
		return string(data), nil
	}
	return c.ExtraPrompt, nil
}
