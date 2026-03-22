package cli

import (
	"context"
	"crypto/tls"
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
	"github.com/secmon-lab/tamamo/pkg/service/tlscert"
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
		pubsubCfg    config.PubSub
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
	flags = append(flags, pubsubCfg.Flags()...)
	flags = append(flags, promptCfg.Flags()...)

	// maskSecret masks a secret value for safe logging.
	// Returns "(set)" if non-empty, "(not set)" if empty.
	maskSecret := func(s string) string {
		if s != "" {
			return "(set)"
		}
		return "(not set)"
	}

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
				s = generated
			}

			// Build emitters
			var emitters []interfaces.Emitter
			emitters = append(emitters, logEmitter.New(logger))
			if we := webhookCfg.Configure(); we != nil {
				emitters = append(emitters, we)
			}
			if pe, err := pubsubCfg.Configure(ctx); err != nil {
				return err
			} else if pe != nil {
				emitters = append(emitters, pe)
				defer func() { _ = pe.Close() }()
			}

			// Log emitter configuration
			logger.Info("webhook emitter config",
				"url", webhookCfg.URL,
				"secret", maskSecret(webhookCfg.Secret),
			)
			logger.Info("pubsub emitter config",
				"project_id", pubsubCfg.ProjectID,
				"topic_id", pubsubCfg.TopicID,
				"sa_key", maskSecret(pubsubCfg.ServiceAccountKey),
			)

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

			// Validate TLS configuration
			if err := serverCfg.ValidateTLS(); err != nil {
				return err
			}

			// Create controller and start server
			srv := honeypotHTTP.New(s, nodeID, emitters)

			logger.Info("starting honeypot server",
				"addr", serverCfg.Addr,
				"tls", serverCfg.TLS,
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

			// Wrap listener with TLS if enabled
			if serverCfg.TLS {
				var cert *tls.Certificate
				if serverCfg.TLSCert != "" {
					// Load user-provided certificate
					loaded, err := tls.LoadX509KeyPair(serverCfg.TLSCert, serverCfg.TLSKey)
					if err != nil {
						return goerr.Wrap(err, "failed to load TLS certificate",
							goerr.V("cert", serverCfg.TLSCert),
							goerr.V("key", serverCfg.TLSKey),
							goerr.T(errutil.TagNotFound),
						)
					}
					cert = &loaded
					logger.Info("using provided TLS certificate",
						"cert", serverCfg.TLSCert,
					)
				} else {
					// Auto-generate self-signed certificate
					generated, err := tlscert.Generate()
					if err != nil {
						return err
					}
					cert = generated
					logger.Info("generated self-signed TLS certificate")
				}

				tlsConfig := &tls.Config{
					Certificates: []tls.Certificate{*cert},
					MinVersion:   tls.VersionTLS12,
				}
				listener = tls.NewListener(listener, tlsConfig)
			}

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
