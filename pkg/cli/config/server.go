package config

import (
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
	"github.com/urfave/cli/v3"
)

// Server holds HTTP server configuration.
type Server struct {
	Addr    string
	NodeID  string
	TLS     bool
	TLSCert string
	TLSKey  string
}

// Flags returns CLI flags for server configuration.
func (c *Server) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "addr",
			Usage:       "Server listen address",
			Value:       "127.0.0.1:8080",
			Destination: &c.Addr,
		},
		&cli.StringFlag{
			Name:        "node-id",
			Usage:       "Honeypot node identifier (default: hostname)",
			Sources:     cli.EnvVars("TAMAMO_NODE_ID"),
			Destination: &c.NodeID,
		},
		&cli.BoolFlag{
			Name:        "tls",
			Usage:       "Enable TLS with auto-generated self-signed certificate",
			Sources:     cli.EnvVars("TAMAMO_TLS"),
			Destination: &c.TLS,
		},
		&cli.StringFlag{
			Name:        "tls-cert",
			Usage:       "Path to TLS certificate file (requires --tls and --tls-key)",
			Sources:     cli.EnvVars("TAMAMO_TLS_CERT"),
			Destination: &c.TLSCert,
		},
		&cli.StringFlag{
			Name:        "tls-key",
			Usage:       "Path to TLS private key file (requires --tls and --tls-cert)",
			Sources:     cli.EnvVars("TAMAMO_TLS_KEY"),
			Destination: &c.TLSKey,
		},
	}
}

// ValidateTLS checks TLS flag combinations and returns an error for invalid configurations.
func (c *Server) ValidateTLS() error {
	// --tls-cert or --tls-key without --tls
	if !c.TLS && (c.TLSCert != "" || c.TLSKey != "") {
		return goerr.New("--tls-cert and --tls-key require --tls to be enabled",
			goerr.T(errutil.TagValidation),
		)
	}
	// Only one of --tls-cert / --tls-key specified
	if c.TLS && (c.TLSCert != "") != (c.TLSKey != "") {
		return goerr.New("--tls-cert and --tls-key must be specified together",
			goerr.T(errutil.TagValidation),
		)
	}
	return nil
}
