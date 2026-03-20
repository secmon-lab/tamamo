package config

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/claude"
	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/m-mizutani/gollem/llm/openai"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
	"github.com/urfave/cli/v3"
)

// LLM holds LLM configuration.
type LLM struct {
	provider   string
	apiKey     string
	model      string
	geminiProject  string
	geminiLocation string
}

// Flags returns CLI flags for LLM configuration.
func (c *LLM) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "llm-provider",
			Usage:       "LLM provider (openai, claude, gemini)",
			Sources:     cli.EnvVars("TAMAMO_LLM_PROVIDER"),
			Destination: &c.provider,
		},
		&cli.StringFlag{
			Name:        "llm-api-key",
			Usage:       "API key for the LLM provider",
			Sources:     cli.EnvVars("TAMAMO_LLM_API_KEY"),
			Destination: &c.apiKey,
		},
		&cli.StringFlag{
			Name:        "llm-model",
			Usage:       "Model name for the LLM provider",
			Sources:     cli.EnvVars("TAMAMO_LLM_MODEL"),
			Destination: &c.model,
		},
		&cli.StringFlag{
			Name:        "gemini-project",
			Usage:       "Google Cloud project ID (required for gemini provider)",
			Sources:     cli.EnvVars("TAMAMO_GEMINI_PROJECT"),
			Destination: &c.geminiProject,
		},
		&cli.StringFlag{
			Name:        "gemini-location",
			Usage:       "Google Cloud location (required for gemini provider)",
			Sources:     cli.EnvVars("TAMAMO_GEMINI_LOCATION"),
			Value:       "us-central1",
			Destination: &c.geminiLocation,
		},
	}
}

// LogConfig logs the LLM configuration.
func (c *LLM) LogConfig(logger *slog.Logger) {
	attrs := []any{
		slog.String("provider", c.provider),
	}
	if c.model != "" {
		attrs = append(attrs, slog.String("model", c.model))
	}
	if c.provider == "gemini" {
		attrs = append(attrs, slog.String("project", c.geminiProject))
		attrs = append(attrs, slog.String("location", c.geminiLocation))
	}
	logger.Info("LLM configuration", attrs...)
}

// Configure creates an LLM client from the configuration.
func (c *LLM) Configure(ctx context.Context) (gollem.LLMClient, error) {
	switch c.provider {
	case "":
		return nil, goerr.New("LLM provider is not configured (--llm-provider or TAMAMO_LLM_PROVIDER)",
			goerr.T(errutil.TagValidation),
		)
	case "openai":
		return c.configureOpenAI(ctx)
	case "claude":
		return c.configureClaude(ctx)
	case "gemini":
		return c.configureGemini(ctx)
	default:
		return nil, goerr.New("unsupported LLM provider",
			goerr.V("provider", c.provider),
			goerr.T(errutil.TagValidation),
		)
	}
}

func (c *LLM) configureOpenAI(ctx context.Context) (gollem.LLMClient, error) {
	if c.apiKey == "" {
		return nil, goerr.New("API key is required for openai provider (--llm-api-key or TAMAMO_LLM_API_KEY)",
			goerr.T(errutil.TagValidation),
		)
	}

	var opts []openai.Option
	if c.model != "" {
		opts = append(opts, openai.WithModel(c.model))
	}

	client, err := openai.New(ctx, c.apiKey, opts...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create OpenAI client",
			goerr.T(errutil.TagExternal),
		)
	}
	return client, nil
}

func (c *LLM) configureClaude(ctx context.Context) (gollem.LLMClient, error) {
	if c.apiKey == "" {
		return nil, goerr.New("API key is required for claude provider (--llm-api-key or TAMAMO_LLM_API_KEY)",
			goerr.T(errutil.TagValidation),
		)
	}

	var opts []claude.Option
	if c.model != "" {
		opts = append(opts, claude.WithModel(c.model))
	}

	client, err := claude.New(ctx, c.apiKey, opts...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create Claude client",
			goerr.T(errutil.TagExternal),
		)
	}
	return client, nil
}

func (c *LLM) configureGemini(ctx context.Context) (gollem.LLMClient, error) {
	if c.geminiProject == "" {
		return nil, goerr.New("Google Cloud project ID is required for gemini provider (--gemini-project or TAMAMO_GEMINI_PROJECT)",
			goerr.T(errutil.TagValidation),
		)
	}

	if !isGCPRegion(c.geminiLocation) {
		return nil, goerr.New("invalid Gemini location: expected a GCP region (e.g., us-central1, asia-northeast1)",
			goerr.V("location", c.geminiLocation),
			goerr.T(errutil.TagValidation),
		)
	}

	var opts []gemini.Option
	if c.model != "" {
		opts = append(opts, gemini.WithModel(c.model))
	}

	client, err := gemini.New(ctx, c.geminiProject, c.geminiLocation, opts...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create Gemini client",
			goerr.V("project", c.geminiProject),
			goerr.V("location", c.geminiLocation),
			goerr.T(errutil.TagExternal),
		)
	}
	return client, nil
}

// isGCPRegion checks if a string looks like a GCP region (e.g., us-central1, europe-west4).
func isGCPRegion(s string) bool {
	if s == "" {
		return false
	}
	if s == "global" {
		return true
	}
	// GCP regions follow pattern: <continent>-<direction><number>
	// Simple heuristic: must contain a hyphen and end with a digit
	if len(s) < 5 {
		return false
	}
	parts := 0
	for _, c := range s {
		if c == '-' {
			parts++
		}
	}
	if parts < 1 {
		return false
	}
	lastChar := s[len(s)-1]
	return lastChar >= '0' && lastChar <= '9'
}
