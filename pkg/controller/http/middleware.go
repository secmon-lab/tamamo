package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/secmon-lab/tamamo/pkg/domain/model/event"
)

// emitMiddleware captures all requests and sends events to emitters.
func (s *Server) emitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture request body for event (limited to 1MB)
		var bodyData any
		if r.Body != nil {
			bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
			if err == nil && len(bodyBytes) > 0 {
				contentType := r.Header.Get("Content-Type")
				switch {
				case strings.HasPrefix(contentType, "application/x-www-form-urlencoded"):
					if values, err := url.ParseQuery(string(bodyBytes)); err == nil {
						parsed := make(map[string]string, len(values))
						for k, v := range values {
							if len(v) > 0 {
								parsed[k] = v[0]
							}
						}
						bodyData = parsed
					} else {
						bodyData = string(bodyBytes)
					}
				default:
					// Try to parse as JSON
					var jsonBody any
					if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
						bodyData = jsonBody
					} else {
						bodyData = string(bodyBytes)
					}
				}

				// Reconstruct body for downstream handlers
				r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
			}
		}

		// Extract headers (limited set for privacy)
		headers := make(map[string]string)
		for _, key := range []string{
			"User-Agent", "Content-Type", "Accept",
			"Accept-Language", "Referer", "Cookie",
			"Authorization", "X-Forwarded-For",
		} {
			if v := r.Header.Get(key); v != "" {
				headers[key] = v
			}
		}

		// Extract source IP
		sourceIP := extractSourceIP(r)

		ev := &event.Event{
			Timestamp: time.Now().UTC(),
			NodeID:    s.nodeID,
			EventType: "http_request",
			SourceIP:  sourceIP,
			Method:    r.Method,
			Path:      r.URL.Path,
			Headers:   headers,
			Body:      bodyData,
			Scenario:  s.scenario.Meta.Name,
		}

		// Emit to all emitters (non-blocking, log errors)
		for _, emitter := range s.emitters {
			if err := emitter.Emit(r.Context(), ev); err != nil {
				slog.Error("failed to emit event",
					slog.String("error", err.Error()),
					slog.String("path", r.URL.Path),
				)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// serverHeadersMiddleware adds the server signature and custom headers from the scenario.
func (s *Server) serverHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.scenario.Meta.ServerSignature != "" {
			w.Header().Set("Server", s.scenario.Meta.ServerSignature)
		}
		for key, value := range s.scenario.Meta.Headers {
			w.Header().Set(key, value)
		}
		next.ServeHTTP(w, r)
	})
}

// extractSourceIP extracts the client IP from the request.
func extractSourceIP(r *http.Request) string {
	// Check X-Forwarded-For first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
