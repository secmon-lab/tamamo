package config

import "github.com/urfave/cli/v3"

// Server holds HTTP server configuration.
type Server struct {
	Addr   string
	NodeID string
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
	}
}
