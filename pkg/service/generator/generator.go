package generator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/tamamo/pkg/domain/interfaces"
	"github.com/secmon-lab/tamamo/pkg/domain/model/scenario"
	"github.com/secmon-lab/tamamo/pkg/service/generator/prompts"
	scenarioSvc "github.com/secmon-lab/tamamo/pkg/service/scenario"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
)

const defaultMaxRetries = 3

// Generator generates honeypot scenarios using an LLM agent.
type Generator struct {
	llmClient  gollem.LLMClient
	outputDir  string
	params     *prompts.Params
	printer    interfaces.Printer
	logger     *slog.Logger
	maxRetries int
}

// Option configures the Generator.
type Option func(*Generator)

// WithPrinter sets the CLI printer for generation progress.
func WithPrinter(p interfaces.Printer) Option {
	return func(g *Generator) {
		g.printer = p
	}
}

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) Option {
	return func(g *Generator) {
		g.logger = l
	}
}

// WithMaxRetries sets the maximum number of retries on validation failure.
func WithMaxRetries(n int) Option {
	return func(g *Generator) {
		g.maxRetries = n
	}
}

// New creates a new Generator.
func New(llmClient gollem.LLMClient, outputDir string, params *prompts.Params, opts ...Option) *Generator {
	g := &Generator{
		llmClient:  llmClient,
		outputDir:  outputDir,
		params:     params,
		printer:    &nopPrinter{},
		logger:     slog.Default(),
		maxRetries: defaultMaxRetries,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Generate runs the LLM agent to create a scenario, retrying on validation failure.
// Agent execution errors (TagExternal, TagGeneration) cause immediate failure.
// Only validation errors (TagValidation) trigger retries.
func (g *Generator) Generate(ctx context.Context) (*scenario.Scenario, error) {
	var lastValidationErr error

	for attempt := range g.maxRetries {
		if attempt > 0 {
			g.logger.Info("retrying scenario generation",
				"attempt", attempt+1,
				"max_retries", g.maxRetries,
				"last_error", lastValidationErr.Error(),
			)
			g.printer.Message("")
			g.printer.Message("🔄 Retrying generation (attempt " + itoa(attempt+1) + "/" + itoa(g.maxRetries) + ")...")
			g.printer.Message("   Previous error: " + lastValidationErr.Error())
		}

		s, err := g.generate(ctx, lastValidationErr)
		if err == nil {
			return s, nil
		}

		// Only retry on validation errors; all other errors fail immediately
		if !goerr.HasTag(err, errutil.TagValidation) {
			return nil, err
		}
		lastValidationErr = err
	}

	return nil, goerr.Wrap(lastValidationErr, "scenario validation failed after all retries",
		goerr.V("max_retries", g.maxRetries),
		goerr.T(errutil.TagValidation),
	)
}

func (g *Generator) generate(ctx context.Context, prevErr error) (*scenario.Scenario, error) {
	prompt, err := prompts.Build(g.params)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build prompt",
			goerr.T(errutil.TagGeneration),
		)
	}

	// On retry, append the previous error to the prompt so the LLM can fix it
	if prevErr != nil {
		prompt += "\n\n## Previous Generation Failed\n\n"
		prompt += "The previous attempt failed validation with the following errors. "
		prompt += "Fix these issues and regenerate ALL files:\n\n"
		prompt += "```\n" + prevErr.Error() + "\n```\n"
	}

	toolSet := newWriteFileToolSet(g.outputDir)
	printer := g.printer

	toolMiddleware := func(next gollem.ToolHandler) gollem.ToolHandler {
		return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
			printer.ToolStart(req.Tool.Name, req.Tool.Arguments)
			resp, err := next(ctx, req)
			if err != nil {
				printer.ToolEnd(req.Tool.Name, nil, err)
				return resp, err
			}
			if resp != nil && resp.Error != nil {
				printer.ToolEnd(req.Tool.Name, resp.Result, resp.Error)
			} else if resp != nil {
				printer.ToolEnd(req.Tool.Name, resp.Result, nil)
			}
			return resp, nil
		}
	}

	contentMiddleware := func(next gollem.ContentBlockHandler) gollem.ContentBlockHandler {
		return func(ctx context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
			resp, err := next(ctx, req)
			if err != nil {
				return resp, err
			}
			if resp != nil {
				for _, text := range resp.Texts {
					if text != "" {
						printer.Message(text)
					}
				}
			}
			return resp, nil
		}
	}

	gollemLogger := slog.New(slog.DiscardHandler)

	agent := gollem.New(g.llmClient,
		gollem.WithSystemPrompt(prompt),
		gollem.WithToolSets(toolSet),
		gollem.WithLogger(gollemLogger),
		gollem.WithToolMiddleware(toolMiddleware),
		gollem.WithContentBlockMiddleware(contentMiddleware),
	)

	if _, err := agent.Execute(ctx, gollem.Text("Generate the honeypot scenario files now.")); err != nil {
		return nil, goerr.Wrap(err, "agent execution failed",
			goerr.T(errutil.TagGeneration),
		)
	}

	s, err := scenarioSvc.Load(ctx, g.outputDir)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to load generated scenario",
			goerr.T(errutil.TagGeneration),
		)
	}

	if err := s.Validate(); err != nil {
		return nil, goerr.Wrap(err, "generated scenario validation failed",
			goerr.T(errutil.TagValidation),
		)
	}

	return s, nil
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

// nopPrinter is a no-op printer for when no printer is configured.
type nopPrinter struct{}

func (p *nopPrinter) ToolStart(_ string, _ map[string]any)        {}
func (p *nopPrinter) ToolEnd(_ string, _ map[string]any, _ error) {}
func (p *nopPrinter) Message(_ string)                            {}
