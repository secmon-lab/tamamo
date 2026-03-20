package config

import (
	"github.com/secmon-lab/tamamo/pkg/domain/interfaces"
	"github.com/secmon-lab/tamamo/pkg/service/emitter/webhook"
	"github.com/urfave/cli/v3"
)

// Webhook holds webhook emitter configuration.
type Webhook struct {
	URL    string
	Secret string
}

// Flags returns CLI flags for webhook configuration.
func (c *Webhook) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "webhook-url",
			Usage:       "Webhook endpoint URL for event notification",
			Sources:     cli.EnvVars("TAMAMO_WEBHOOK_URL"),
			Destination: &c.URL,
		},
		&cli.StringFlag{
			Name:        "webhook-secret",
			Usage:       "Webhook HMAC-SHA256 signing secret",
			Sources:     cli.EnvVars("TAMAMO_WEBHOOK_SECRET"),
			Destination: &c.Secret,
		},
	}
}

// Configure creates a WebhookEmitter if URL is set, returns nil otherwise.
func (c *Webhook) Configure() interfaces.Emitter {
	if c.URL == "" {
		return nil
	}
	var opts []webhook.Option
	if c.Secret != "" {
		opts = append(opts, webhook.WithSecret(c.Secret))
	}
	return webhook.New(c.URL, opts...)
}
