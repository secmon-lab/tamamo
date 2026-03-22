package pubsub

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/pubsub/v2"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/tamamo/pkg/domain/model/event"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
	"google.golang.org/api/option"
)

// Emitter sends events to Google Cloud Pub/Sub.
type Emitter struct {
	client    *pubsub.Client
	publisher *pubsub.Publisher
}

// New creates a Pub/Sub Emitter with the given project and topic IDs.
// Optional client options (e.g. option.WithCredentialsJSON) can be passed.
func New(ctx context.Context, projectID, topicID string, opts ...option.ClientOption) (*Emitter, error) {
	client, err := pubsub.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create pubsub client",
			goerr.V("project_id", projectID),
			goerr.T(errutil.TagExternal),
		)
	}

	return &Emitter{
		client:    client,
		publisher: client.Publisher(topicID),
	}, nil
}

// NewWithClient creates a Pub/Sub Emitter with an existing client.
// This is intended for testing with fake Pub/Sub servers.
func NewWithClient(client *pubsub.Client, topicID string) *Emitter {
	return &Emitter{
		client:    client,
		publisher: client.Publisher(topicID),
	}
}

// Emit publishes the event as JSON to the Pub/Sub topic.
func (e *Emitter) Emit(ctx context.Context, ev *event.Event) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return goerr.Wrap(err, "failed to marshal event for pubsub",
			goerr.T(errutil.TagInternal),
		)
	}

	result := e.publisher.Publish(ctx, &pubsub.Message{
		Data: data,
	})

	if _, err := result.Get(ctx); err != nil {
		return goerr.Wrap(err, "failed to publish event to pubsub",
			goerr.T(errutil.TagExternal),
		)
	}

	return nil
}

// Close stops the publisher and closes the Pub/Sub client.
func (e *Emitter) Close() error {
	e.publisher.Stop()
	if err := e.client.Close(); err != nil {
		return goerr.Wrap(err, "failed to close pubsub client",
			goerr.T(errutil.TagExternal),
		)
	}
	return nil
}
