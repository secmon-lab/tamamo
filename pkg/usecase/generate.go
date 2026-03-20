package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/tamamo/pkg/domain/model/scenario"
	"github.com/secmon-lab/tamamo/pkg/service/generator"
	"github.com/secmon-lab/tamamo/pkg/service/generator/prompts"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
)

// GenerateOption holds parameters for scenario generation.
type GenerateOption struct {
	SiteType   string
	Style      string
	Taste      string
	Lang       string
	Extra      string
	MaxRetries int
}

// Generate creates a new honeypot scenario using the LLM.
// outputDir is the directory where the LLM agent writes files.
func (u *UseCase) Generate(ctx context.Context, outputDir string, opt GenerateOption) (*scenario.Scenario, error) {
	if u.llmClient == nil {
		return nil, goerr.New("LLM client is required for generation",
			goerr.T(errutil.TagValidation),
		)
	}

	params := &prompts.Params{
		SiteType:    opt.SiteType,
		Style:       opt.Style,
		Taste:       opt.Taste,
		Lang:        opt.Lang,
		ExtraPrompt: opt.Extra,
	}

	genOpts := []generator.Option{
		generator.WithPrinter(u.printer),
		generator.WithLogger(u.logger),
	}
	if opt.MaxRetries > 0 {
		genOpts = append(genOpts, generator.WithMaxRetries(opt.MaxRetries))
	}

	gen := generator.New(
		u.llmClient,
		outputDir,
		params,
		genOpts...,
	)

	s, err := gen.Generate(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "scenario generation failed",
			goerr.T(errutil.TagGeneration),
		)
	}

	return s, nil
}
