package http

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/secmon-lab/tamamo/pkg/domain/model/scenario"
)

// newRouteHandler creates an HTTP handler for a scenario route.
func newRouteHandler(route scenario.Route) http.HandlerFunc {
	if route.Hang {
		return newHangHandler()
	}
	if route.Auth != nil {
		return newAuthRouteHandler(route)
	}
	return newStaticRouteHandler(route)
}

// newHangHandler returns a handler that never sends a response.
// The connection stays open until the client disconnects or the server shuts down.
func newHangHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}
}

// newStaticRouteHandler returns a handler that always returns the same response.
func newStaticRouteHandler(route scenario.Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeRouteResponse(w, route.StatusCode, route.Headers, route.Body)
	}
}

// authState tracks unique credential submissions per source IP.
type authState struct {
	// results caches the success/failure outcome for each credential hash.
	// true = success, false = failure. Once set, the result never changes.
	results map[string]bool
	// failures counts the number of unique credentials that resulted in failure.
	failures int
}

// newAuthRouteHandler returns a handler that tracks login attempts per source IP.
// The first MinFailures unique credentials always fail. After that, each new
// unique credential succeeds with SuccessProbability chance (determined
// deterministically from the credential hash). Same credentials always return
// the same result.
func newAuthRouteHandler(route scenario.Route) http.HandlerFunc {
	var (
		mu       sync.Mutex
		attempts = make(map[string]*authState)
	)
	auth := route.Auth

	return func(w http.ResponseWriter, r *http.Request) {
		sourceIP := extractSourceIP(r)
		credHash := computeCredentialHash(r, auth.CredentialFields)

		mu.Lock()
		state, ok := attempts[sourceIP]
		if !ok {
			state = &authState{results: make(map[string]bool)}
			attempts[sourceIP] = state
		}

		success, seen := state.results[credHash]
		if !seen {
			if state.failures < auth.MinFailures {
				// Must fail: haven't reached minimum failures yet
				success = false
			} else {
				// Determine success probabilistically from credential hash
				success = hashProbability(credHash) < auth.SuccessProbability
			}
			state.results[credHash] = success
			if !success {
				state.failures++
			}
		}
		mu.Unlock()

		if !success {
			writeRouteResponse(w, auth.FailureStatusCode, auth.FailureHeaders, auth.FailureBody)
			return
		}

		writeRouteResponse(w, route.StatusCode, route.Headers, route.Body)
	}
}

// hashProbability derives a deterministic probability value (0.0-1.0) from a hex hash string.
// This ensures the same credential always produces the same outcome.
func hashProbability(hexHash string) float64 {
	// Use first 8 bytes of hash to derive a float64 in [0, 1)
	raw, err := hex.DecodeString(hexHash[:16])
	if err != nil {
		return 0
	}
	v := binary.BigEndian.Uint64(raw)
	return float64(v) / float64(^uint64(0))
}

// computeCredentialHash reads the request body, extracts the credential fields,
// and returns a SHA-256 hash of the sorted key=value pairs.
// If credentialFields is empty, the entire body is hashed as a fallback.
func computeCredentialHash(r *http.Request, credentialFields []string) string {
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil || len(bodyBytes) == 0 {
		return hashString("")
	}
	// Reconstruct body for downstream use (event emission already captured it)
	r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	if len(credentialFields) == 0 {
		return hashString(string(bodyBytes))
	}

	// Parse body into field values based on Content-Type
	fields := parseBodyFields(r.Header.Get("Content-Type"), bodyBytes)
	if fields == nil {
		return hashString(string(bodyBytes))
	}

	// Extract only credential fields, sorted for deterministic hashing
	sorted := make([]string, len(credentialFields))
	copy(sorted, credentialFields)
	sort.Strings(sorted)

	var parts []string
	for _, key := range sorted {
		parts = append(parts, fmt.Sprintf("%s=%s", key, fields[key]))
	}
	return hashString(strings.Join(parts, "\n"))
}

// parseBodyFields parses the request body into a string map based on content type.
func parseBodyFields(contentType string, body []byte) map[string]string {
	switch {
	case strings.HasPrefix(contentType, "application/x-www-form-urlencoded"):
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil
		}
		result := make(map[string]string, len(values))
		for k, v := range values {
			if len(v) > 0 {
				result[k] = v[0]
			}
		}
		return result

	default:
		// Try JSON
		var jsonMap map[string]any
		if err := json.Unmarshal(body, &jsonMap); err != nil {
			return nil
		}
		result := make(map[string]string, len(jsonMap))
		for k, v := range jsonMap {
			result[k] = fmt.Sprintf("%v", v)
		}
		return result
	}
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func writeRouteResponse(w http.ResponseWriter, statusCode int, headers map[string]string, body string) {
	for key, value := range headers {
		w.Header().Set(key, value)
	}
	w.WriteHeader(statusCode)
	if body != "" {
		_, _ = w.Write([]byte(body))
	}
}
