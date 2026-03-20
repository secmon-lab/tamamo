package interfaces

import (
	"context"

	"github.com/secmon-lab/tamamo/pkg/domain/model/event"
)

// Emitter abstracts event notification for honeypot interactions.
// Implementations include log output, webhook, and future extensions (Pub/Sub, SQS, etc.).
type Emitter interface {
	Emit(ctx context.Context, ev *event.Event) error
}
