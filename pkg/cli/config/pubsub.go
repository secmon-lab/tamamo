package config

import (
	"context"

	psEmitter "github.com/secmon-lab/tamamo/pkg/service/emitter/pubsub"
	"github.com/urfave/cli/v3"
	"google.golang.org/api/option"
)

// PubSub holds Google Cloud Pub/Sub emitter configuration.
type PubSub struct {
	ProjectID         string
	TopicID           string
	ServiceAccountKey string
}

// Flags returns CLI flags for Pub/Sub configuration.
func (c *PubSub) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "pubsub-project-id",
			Usage:       "Google Cloud project ID for Pub/Sub",
			Sources:     cli.EnvVars("TAMAMO_PUBSUB_PROJECT_ID"),
			Destination: &c.ProjectID,
		},
		&cli.StringFlag{
			Name:        "pubsub-topic-id",
			Usage:       "Pub/Sub topic ID for event notification",
			Sources:     cli.EnvVars("TAMAMO_PUBSUB_TOPIC_ID"),
			Destination: &c.TopicID,
		},
		&cli.StringFlag{
			Name:        "pubsub-sa-key",
			Usage:       "Service account key JSON data for Pub/Sub authentication (takes priority over ADC)",
			Sources:     cli.EnvVars("TAMAMO_PUBSUB_SA_KEY"),
			Destination: &c.ServiceAccountKey,
		},
	}
}

// Configure creates a PubSubEmitter if both ProjectID and TopicID are set, returns nil otherwise.
// When ServiceAccountKey is provided, it is used for authentication instead of ADC.
func (c *PubSub) Configure(ctx context.Context) (*psEmitter.Emitter, error) {
	if c.ProjectID == "" || c.TopicID == "" {
		return nil, nil
	}

	var opts []option.ClientOption
	if c.ServiceAccountKey != "" {
		opts = append(opts, option.WithAuthCredentialsJSON(option.ServiceAccount, []byte(c.ServiceAccountKey)))
	}

	return psEmitter.New(ctx, c.ProjectID, c.TopicID, opts...)
}
