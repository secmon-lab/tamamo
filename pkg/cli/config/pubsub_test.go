package config_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/tamamo/pkg/cli/config"
	"github.com/urfave/cli/v3"
)

func TestPubSubConfigure(t *testing.T) {
	t.Run("returns nil when project ID is empty", func(t *testing.T) {
		cfg := config.PubSub{TopicID: "my-topic"}
		emitter, err := cfg.Configure(context.Background())
		gt.NoError(t, err)
		gt.V(t, emitter).Nil()
	})

	t.Run("returns nil when topic ID is empty", func(t *testing.T) {
		cfg := config.PubSub{ProjectID: "my-project"}
		emitter, err := cfg.Configure(context.Background())
		gt.NoError(t, err)
		gt.V(t, emitter).Nil()
	})

	t.Run("returns nil when both are empty", func(t *testing.T) {
		cfg := config.PubSub{}
		emitter, err := cfg.Configure(context.Background())
		gt.NoError(t, err)
		gt.V(t, emitter).Nil()
	})

	t.Run("returns nil when project ID is empty string and topic is set", func(t *testing.T) {
		cfg := config.PubSub{ProjectID: "", TopicID: "events"}
		emitter, err := cfg.Configure(context.Background())
		gt.NoError(t, err)
		gt.V(t, emitter).Nil()
	})

	t.Run("returns nil when topic ID is empty string and project is set", func(t *testing.T) {
		cfg := config.PubSub{ProjectID: "my-project", TopicID: ""}
		emitter, err := cfg.Configure(context.Background())
		gt.NoError(t, err)
		gt.V(t, emitter).Nil()
	})

	t.Run("returns nil when only sa-key is set without project and topic", func(t *testing.T) {
		cfg := config.PubSub{ServiceAccountKey: `{"type":"service_account"}`}
		emitter, err := cfg.Configure(context.Background())
		gt.NoError(t, err)
		gt.V(t, emitter).Nil()
	})

	t.Run("returns error with invalid sa-key when project and topic are set", func(t *testing.T) {
		cfg := config.PubSub{
			ProjectID:         "my-project",
			TopicID:           "my-topic",
			ServiceAccountKey: "not-valid-json",
		}
		emitter, err := cfg.Configure(context.Background())
		gt.V(t, err).NotNil()
		gt.V(t, emitter).Nil()
	})
}

func TestPubSubFlags(t *testing.T) {
	t.Run("returns three flags", func(t *testing.T) {
		cfg := config.PubSub{}
		flags := cfg.Flags()
		gt.A(t, flags).Length(3)
	})

	t.Run("flag names are correct", func(t *testing.T) {
		cfg := config.PubSub{}
		flags := cfg.Flags()

		names := make([]string, 0, len(flags))
		for _, f := range flags {
			names = append(names, f.Names()[0])
		}
		gt.A(t, names).Length(3)
		gt.A(t, names).Has("pubsub-project-id")
		gt.A(t, names).Has("pubsub-topic-id")
		gt.A(t, names).Has("pubsub-sa-key")
	})

	t.Run("flags bind to destination fields", func(t *testing.T) {
		cfg := config.PubSub{}
		flags := cfg.Flags()

		for _, f := range flags {
			sf, ok := f.(*cli.StringFlag)
			gt.V(t, ok).Equal(true)
			gt.V(t, sf.Destination).NotNil()
		}
	})
}
