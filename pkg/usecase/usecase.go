package usecase

import (
	"log/slog"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/tamamo/pkg/domain/interfaces"
)

// UseCase orchestrates application operations.
type UseCase struct {
	llmClient gollem.LLMClient
	printer   interfaces.Printer
	logger    *slog.Logger
}

// Option configures the UseCase.
type Option func(*UseCase)

// WithLLMClient sets the LLM client.
func WithLLMClient(c gollem.LLMClient) Option {
	return func(u *UseCase) {
		u.llmClient = c
	}
}

// WithPrinter sets the CLI printer.
func WithPrinter(p interfaces.Printer) Option {
	return func(u *UseCase) {
		u.printer = p
	}
}

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) Option {
	return func(u *UseCase) {
		u.logger = l
	}
}

// New creates a new UseCase.
func New(opts ...Option) *UseCase {
	u := &UseCase{
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}
