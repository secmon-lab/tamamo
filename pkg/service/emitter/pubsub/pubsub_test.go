package pubsub_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"cloud.google.com/go/pubsub/v2/pstest"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/tamamo/pkg/domain/model/event"
	psEmitter "github.com/secmon-lab/tamamo/pkg/service/emitter/pubsub"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func setupFake(t *testing.T) (*pstest.Server, *pubsub.Client) {
	t.Helper()
	srv := pstest.NewServer()
	t.Cleanup(func() { _ = srv.Close() })

	ctx := context.Background()
	conn, err := grpc.NewClient(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	gt.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	client, err := pubsub.NewClient(ctx, "test-project", option.WithGRPCConn(conn))
	gt.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	return srv, client
}

func createTopic(t *testing.T, client *pubsub.Client, topicID string) {
	t.Helper()
	ctx := context.Background()
	_, err := client.TopicAdminClient.CreateTopic(ctx, &pubsubpb.Topic{
		Name: "projects/test-project/topics/" + topicID,
	})
	gt.NoError(t, err)
}

func TestEmit(t *testing.T) {
	t.Run("publishes event with all fields as JSON", func(t *testing.T) {
		srv, client := setupFake(t)
		ctx := context.Background()
		createTopic(t, client, "test-topic")

		emitter := psEmitter.NewWithClient(client, "test-topic")
		t.Cleanup(func() { _ = emitter.Close() })

		ts := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
		ev := &event.Event{
			Timestamp: ts,
			NodeID:    "test-node",
			EventType: "http_request",
			SourceIP:  "192.168.1.1",
			Method:    "POST",
			Path:      "/api/auth/login",
			Headers:   map[string]string{"User-Agent": "curl/8.0", "Content-Type": "application/json"},
			Body:      map[string]string{"username": "admin", "password": "test"},
			Scenario:  "test-scenario",
		}

		err := emitter.Emit(ctx, ev)
		gt.NoError(t, err)

		msgs := srv.Messages()
		gt.A(t, msgs).Length(1)

		var received event.Event
		gt.NoError(t, json.Unmarshal(msgs[0].Data, &received))
		gt.Equal(t, received.NodeID, "test-node")
		gt.Equal(t, received.EventType, "http_request")
		gt.Equal(t, received.SourceIP, "192.168.1.1")
		gt.Equal(t, received.Method, "POST")
		gt.Equal(t, received.Path, "/api/auth/login")
		gt.Equal(t, received.Scenario, "test-scenario")
		gt.Equal(t, received.Timestamp, ts)
		gt.V(t, received.Headers).NotNil()
		gt.Equal(t, received.Headers["User-Agent"], "curl/8.0")
		gt.Equal(t, received.Headers["Content-Type"], "application/json")
	})

	t.Run("publishes event with nil body", func(t *testing.T) {
		srv, client := setupFake(t)
		ctx := context.Background()
		createTopic(t, client, "test-topic")

		emitter := psEmitter.NewWithClient(client, "test-topic")
		t.Cleanup(func() { _ = emitter.Close() })

		ev := &event.Event{
			Timestamp: time.Now(),
			NodeID:    "node-1",
			EventType: "http_request",
			SourceIP:  "10.0.0.1",
			Method:    "GET",
			Path:      "/health",
			Scenario:  "scenario-1",
		}

		err := emitter.Emit(ctx, ev)
		gt.NoError(t, err)

		msgs := srv.Messages()
		gt.A(t, msgs).Length(1)

		var received event.Event
		gt.NoError(t, json.Unmarshal(msgs[0].Data, &received))
		gt.Equal(t, received.Method, "GET")
		gt.Equal(t, received.Path, "/health")
		gt.V(t, received.Body).Nil()
	})

	t.Run("publishes event with empty headers", func(t *testing.T) {
		srv, client := setupFake(t)
		ctx := context.Background()
		createTopic(t, client, "test-topic")

		emitter := psEmitter.NewWithClient(client, "test-topic")
		t.Cleanup(func() { _ = emitter.Close() })

		ev := &event.Event{
			Timestamp: time.Now(),
			EventType: "http_request",
			Headers:   map[string]string{},
		}

		err := emitter.Emit(ctx, ev)
		gt.NoError(t, err)

		msgs := srv.Messages()
		gt.A(t, msgs).Length(1)

		var received event.Event
		gt.NoError(t, json.Unmarshal(msgs[0].Data, &received))
		gt.V(t, received.Headers).NotNil()
		gt.V(t, len(received.Headers)).Equal(0)
	})

	t.Run("publishes multiple events sequentially", func(t *testing.T) {
		srv, client := setupFake(t)
		ctx := context.Background()
		createTopic(t, client, "test-topic")

		emitter := psEmitter.NewWithClient(client, "test-topic")
		t.Cleanup(func() { _ = emitter.Close() })

		paths := []string{"/path/a", "/path/b", "/path/c"}
		for _, p := range paths {
			ev := &event.Event{
				Timestamp: time.Now(),
				EventType: "http_request",
				Path:      p,
			}
			gt.NoError(t, emitter.Emit(ctx, ev))
		}

		msgs := srv.Messages()
		gt.A(t, msgs).Length(3)

		receivedPaths := make([]string, 0, 3)
		for _, msg := range msgs {
			var received event.Event
			gt.NoError(t, json.Unmarshal(msg.Data, &received))
			receivedPaths = append(receivedPaths, received.Path)
		}
		gt.A(t, receivedPaths).Length(3)
		gt.A(t, receivedPaths).Has("/path/a")
		gt.A(t, receivedPaths).Has("/path/b")
		gt.A(t, receivedPaths).Has("/path/c")
	})

	t.Run("returns error when context is cancelled", func(t *testing.T) {
		_, client := setupFake(t)
		createTopic(t, client, "test-topic")

		emitter := psEmitter.NewWithClient(client, "test-topic")
		t.Cleanup(func() { _ = emitter.Close() })

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		ev := &event.Event{
			Timestamp: time.Now(),
			EventType: "http_request",
		}

		err := emitter.Emit(ctx, ev)
		gt.V(t, err).NotNil()
	})

	t.Run("published JSON is valid and round-trips correctly", func(t *testing.T) {
		srv, client := setupFake(t)
		ctx := context.Background()
		createTopic(t, client, "test-topic")

		emitter := psEmitter.NewWithClient(client, "test-topic")
		t.Cleanup(func() { _ = emitter.Close() })

		original := &event.Event{
			Timestamp: time.Date(2026, 6, 15, 8, 30, 0, 0, time.UTC),
			NodeID:    "honeypot-us-east-1",
			EventType: "http_request",
			SourceIP:  "203.0.113.42",
			Method:    "PUT",
			Path:      "/api/v2/users/settings",
			Headers: map[string]string{
				"User-Agent":    "Mozilla/5.0",
				"Authorization": "Bearer eyJhbGciOi...",
			},
			Body:     map[string]any{"key": "value", "nested": map[string]any{"a": float64(1)}},
			Scenario: "enterprise-admin-panel",
		}

		gt.NoError(t, emitter.Emit(ctx, original))

		msgs := srv.Messages()
		gt.A(t, msgs).Length(1)

		// Verify the data is valid JSON
		gt.V(t, json.Valid(msgs[0].Data)).Equal(true)

		var received event.Event
		gt.NoError(t, json.Unmarshal(msgs[0].Data, &received))

		gt.Equal(t, received.Timestamp, original.Timestamp)
		gt.Equal(t, received.NodeID, original.NodeID)
		gt.Equal(t, received.EventType, original.EventType)
		gt.Equal(t, received.SourceIP, original.SourceIP)
		gt.Equal(t, received.Method, original.Method)
		gt.Equal(t, received.Path, original.Path)
		gt.Equal(t, received.Scenario, original.Scenario)
		gt.Equal(t, received.Headers["User-Agent"], "Mozilla/5.0")
		gt.Equal(t, received.Headers["Authorization"], "Bearer eyJhbGciOi...")
	})
}

func TestClose(t *testing.T) {
	t.Run("closes without error", func(t *testing.T) {
		_, client := setupFake(t)
		createTopic(t, client, "test-topic")

		emitter := psEmitter.NewWithClient(client, "test-topic")
		err := emitter.Close()
		gt.NoError(t, err)
	})

	t.Run("close after emit succeeds", func(t *testing.T) {
		_, client := setupFake(t)
		ctx := context.Background()
		createTopic(t, client, "test-topic")

		emitter := psEmitter.NewWithClient(client, "test-topic")

		ev := &event.Event{
			Timestamp: time.Now(),
			EventType: "http_request",
			Path:      "/test",
		}
		gt.NoError(t, emitter.Emit(ctx, ev))

		err := emitter.Close()
		gt.NoError(t, err)
	})
}
