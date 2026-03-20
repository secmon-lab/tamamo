package http

import (
	"net/http"
	"sync"

	"github.com/secmon-lab/tamamo/pkg/domain/model/scenario"
)

// newRouteHandler creates an HTTP handler for a scenario route.
func newRouteHandler(route scenario.Route) http.HandlerFunc {
	if route.Auth != nil {
		return newAuthRouteHandler(route)
	}
	return newStaticRouteHandler(route)
}

// newStaticRouteHandler returns a handler that always returns the same response.
func newStaticRouteHandler(route scenario.Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeRouteResponse(w, route.StatusCode, route.Headers, route.Body)
	}
}

// newAuthRouteHandler returns a handler that tracks login attempts per source IP
// and returns failure responses until the configured threshold is reached.
func newAuthRouteHandler(route scenario.Route) http.HandlerFunc {
	var (
		mu       sync.Mutex
		attempts = make(map[string]int)
	)
	auth := route.Auth

	return func(w http.ResponseWriter, r *http.Request) {
		sourceIP := extractSourceIP(r)

		mu.Lock()
		count := attempts[sourceIP]
		attempts[sourceIP] = count + 1
		mu.Unlock()

		if count < auth.FailuresBeforeSuccess {
			// Return failure response
			writeRouteResponse(w, auth.FailureStatusCode, auth.FailureHeaders, auth.FailureBody)
			return
		}

		// Return success response
		writeRouteResponse(w, route.StatusCode, route.Headers, route.Body)
	}
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
