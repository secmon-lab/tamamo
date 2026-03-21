package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/secmon-lab/tamamo/pkg/cli/config"
	scenarioSvc "github.com/secmon-lab/tamamo/pkg/service/scenario"
	"github.com/secmon-lab/tamamo/pkg/usecase"
	"github.com/urfave/cli/v3"
)

func newGenerateCommand() *cli.Command {
	var (
		llmCfg     config.LLM
		logCfg     config.Logger
		promptCfg  config.Prompt
		output     string
		dumpDir    string
		keepTmp    bool
		maxRetries int
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "output",
			Aliases:     []string{"o"},
			Usage:       "Output ZIP file path for generated scenario",
			Value:       "scenario.zip",
			Destination: &output,
		},
		&cli.StringFlag{
			Name:        "dump-dir",
			Usage:       "Dump raw generated files to this directory (for debugging)",
			Destination: &dumpDir,
		},
		&cli.BoolFlag{
			Name:        "keep-tmp",
			Usage:       "Keep temporary generation directory (for debugging)",
			Destination: &keepTmp,
		},
		&cli.IntFlag{
			Name:        "max-retries",
			Usage:       "Maximum retries on validation failure",
			Value:       3,
			Destination: &maxRetries,
		},
	}
	flags = append(flags, llmCfg.Flags()...)
	flags = append(flags, logCfg.Flags()...)
	flags = append(flags, promptCfg.Flags()...)

	return &cli.Command{
		Name:    "generate",
		Aliases: []string{"g"},
		Usage:   "Generate honeypot scenario data using LLM",
		Flags:   flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			logger := logCfg.Configure()
			llmCfg.LogConfig(logger)
			promptCfg.LogConfig(logger)

			llmClient, err := llmCfg.Configure(ctx)
			if err != nil {
				return err
			}

			extra, err := promptCfg.ResolveExtraPrompt()
			if err != nil {
				return err
			}

			// Create temp directory for generation
			tmpDir, err := os.MkdirTemp("", "tamamo-scenario-*")
			if err != nil {
				return fmt.Errorf("failed to create temp directory: %w", err)
			}
			if keepTmp {
				fmt.Printf("   Temp dir: %s\n", tmpDir)
			} else {
				defer func() { _ = os.RemoveAll(tmpDir) }()
			}

			uc := usecase.New(
				usecase.WithLLMClient(llmClient),
				usecase.WithPrinter(newCLIPrinter()),
				usecase.WithLogger(logger),
			)

			s, err := uc.Generate(ctx, tmpDir, usecase.GenerateOption{
				SiteType:    promptCfg.SiteType,
				Style:       promptCfg.Style,
				Taste:       promptCfg.Taste,
				Layout:      promptCfg.Layout,
				ColorScheme: promptCfg.ColorScheme,
				Lang:        promptCfg.Lang,
				Extra:       extra,
				MaxRetries:  maxRetries,
			})
			if err != nil {
				return err
			}

			// Export as ZIP
			if err := scenarioSvc.SaveAsZip(tmpDir, output); err != nil {
				return fmt.Errorf("failed to export scenario as ZIP: %w", err)
			}

			// Dump raw files if requested
			if dumpDir != "" {
				if err := scenarioSvc.Save(ctx, dumpDir, s); err != nil {
					return fmt.Errorf("failed to dump scenario files: %w", err)
				}
			}

			fmt.Printf("✅ Scenario generated: %q\n", s.Meta.Name)
			fmt.Printf("   Output: %s\n", output)
			return nil
		},
	}
}
