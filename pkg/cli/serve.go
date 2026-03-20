package cli

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/tamamo/pkg/cli/config"
	honeypotHTTP "github.com/secmon-lab/tamamo/pkg/controller/http"
	"github.com/secmon-lab/tamamo/pkg/domain/interfaces"
	"github.com/secmon-lab/tamamo/pkg/domain/model/scenario"
	logEmitter "github.com/secmon-lab/tamamo/pkg/service/emitter/log"
	"github.com/secmon-lab/tamamo/pkg/usecase"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
	"github.com/urfave/cli/v3"
)

func newServeCommand() *cli.Command {
	var (
		llmCfg       config.LLM
		logCfg       config.Logger
		serverCfg    config.Server
		webhookCfg   config.Webhook
		promptCfg    config.Prompt
		scenarioPath string
		keepTmp      bool
		maxRetries   int
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "scenario",
			Aliases:     []string{"s"},
			Usage:       "Path to scenario directory or ZIP file",
			Destination: &scenarioPath,
		},
		&cli.BoolFlag{
			Name:        "keep-tmp",
			Usage:       "Keep temporary generation directory (for debugging)",
			Destination: &keepTmp,
		},
		&cli.IntFlag{
			Name:        "max-retries",
			Usage:       "Maximum retries on generation validation failure",
			Value:       3,
			Destination: &maxRetries,
		},
	}
	flags = append(flags, llmCfg.Flags()...)
	flags = append(flags, logCfg.Flags()...)
	flags = append(flags, serverCfg.Flags()...)
	flags = append(flags, webhookCfg.Flags()...)
	flags = append(flags, promptCfg.Flags()...)

	return &cli.Command{
		Name:    "serve",
		Aliases: []string{"s"},
		Usage:   "Start honeypot HTTP server",
		Flags:   flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			logger := logCfg.Configure()
			slog.SetDefault(logger)

			var s *scenario.Scenario

			if scenarioPath != "" {
				logger.Info("loading scenario", "path", scenarioPath)
				// Load existing scenario
				uc := usecase.New(usecase.WithLogger(logger))
				loaded, err := uc.LoadScenario(ctx, scenarioPath)
				if err != nil {
					return err
				}
				s = loaded
			} else {
				// Auto-generate into temp directory
				llmCfg.LogConfig(logger)
				promptCfg.LogConfig(logger)
				llmClient, err := llmCfg.Configure(ctx)
				if err != nil {
					return err
				}

				tmpDir, err := os.MkdirTemp("", "tamamo-serve-*")
				if err != nil {
					return fmt.Errorf("failed to create temp directory: %w", err)
				}
				if keepTmp {
					fmt.Printf("   Temp dir: %s\n", tmpDir)
				} else {
					defer func() { _ = os.RemoveAll(tmpDir) }()
				}

				extra, err := promptCfg.ResolveExtraPrompt()
				if err != nil {
					return err
				}

				uc := usecase.New(
					usecase.WithLLMClient(llmClient),
					usecase.WithPrinter(newCLIPrinter()),
					usecase.WithLogger(logger),
				)
				generated, err := uc.Generate(ctx, tmpDir, usecase.GenerateOption{
					SiteType:   promptCfg.SiteType,
					Style:      promptCfg.Style,
					Taste:      promptCfg.Taste,
					Layout:     promptCfg.Layout,
					Lang:       promptCfg.Lang,
					Extra:      extra,
					MaxRetries: maxRetries,
				})
				if err != nil {
					return err
				}
				s = generated
			}

			// Build emitters
			var emitters []interfaces.Emitter
			emitters = append(emitters, logEmitter.New(logger))
			if we := webhookCfg.Configure(); we != nil {
				emitters = append(emitters, we)
			}

			// Resolve node ID
			nodeID := serverCfg.NodeID
			if nodeID == "" {
				h, _ := os.Hostname()
				if h != "" {
					nodeID = h
				} else {
					nodeID = "tamamo-node"
				}
			}

			// Create controller and start server
			srv := honeypotHTTP.New(s, nodeID, emitters)

			logger.Info("starting honeypot server",
				"addr", serverCfg.Addr,
				"scenario", s.Meta.Name,
				"node_id", nodeID,
			)

			listener, err := net.Listen("tcp", serverCfg.Addr)
			if err != nil {
				return goerr.Wrap(err, "failed to listen",
					goerr.V("addr", serverCfg.Addr),
					goerr.T(errutil.TagInternal),
				)
			}
			defer func() { _ = listener.Close() }()

			server := &http.Server{
				Handler:           srv.Handler(),
				ReadHeaderTimeout: 10 * time.Second,
			}

			go func() {
				<-ctx.Done()
				_ = server.Shutdown(context.Background())
			}()

			if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
				return goerr.Wrap(err, "server error",
					goerr.T(errutil.TagInternal),
				)
			}

			return nil
		},
	}
}
