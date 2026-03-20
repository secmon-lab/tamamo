package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/tamamo/pkg/domain/model/event"
	"github.com/secmon-lab/tamamo/pkg/utils/errutil"
)

// Emitter sends events via HTTP POST with optional HMAC-SHA256 signature.
type Emitter struct {
	url    string
	secret string
	client *http.Client
}

// Option configures the webhook Emitter.
type Option func(*Emitter)

// WithSecret sets the HMAC-SHA256 signing secret.
func WithSecret(secret string) Option {
	return func(e *Emitter) {
		e.secret = secret
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(e *Emitter) {
		e.client.Timeout = d
	}
}

// New creates a webhook Emitter with the given URL.
func New(url string, opts ...Option) *Emitter {
	e := &Emitter{
		url: url,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Emit sends the event as a JSON POST. If a secret is configured, adds HMAC-SHA256 signature.
func (e *Emitter) Emit(ctx context.Context, ev *event.Event) error {
	body, err := json.Marshal(ev)
	if err != nil {
		return goerr.Wrap(err, "failed to marshal event for webhook",
			goerr.T(errutil.TagInternal),
		)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url, bytes.NewReader(body))
	if err != nil {
		return goerr.Wrap(err, "failed to create webhook request",
			goerr.V("url", e.url),
			goerr.T(errutil.TagExternal),
		)
	}
	req.Header.Set("Content-Type", "application/json")

	if e.secret != "" {
		mac := hmac.New(sha256.New, []byte(e.secret))
		mac.Write(body)
		sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Tamamo-Signature", sig)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return goerr.Wrap(err, "failed to send webhook",
			goerr.V("url", e.url),
			goerr.T(errutil.TagExternal),
		)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return goerr.New(fmt.Sprintf("webhook returned non-2xx status: %d", resp.StatusCode),
			goerr.V("url", e.url),
			goerr.V("status", resp.StatusCode),
			goerr.T(errutil.TagExternal),
		)
	}

	return nil
}
