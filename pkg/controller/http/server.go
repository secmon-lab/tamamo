package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/secmon-lab/tamamo/pkg/domain/interfaces"
	"github.com/secmon-lab/tamamo/pkg/domain/model/scenario"
)

// Server is the honeypot HTTP server.
type Server struct {
	router   *chi.Mux
	scenario *scenario.Scenario
	emitters []interfaces.Emitter
	nodeID   string
}

// New creates a new honeypot HTTP server.
func New(s *scenario.Scenario, nodeID string, emitters []interfaces.Emitter) *Server {
	srv := &Server{
		router:   chi.NewRouter(),
		scenario: s,
		emitters: emitters,
		nodeID:   nodeID,
	}
	srv.setupRoutes()
	return srv
}

// Handler returns the http.Handler for the server.
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) setupRoutes() {
	// Add event emission middleware
	s.router.Use(s.emitMiddleware)

	// Add server signature headers
	s.router.Use(s.serverHeadersMiddleware)

	// Register scenario-defined routes
	for _, route := range s.scenario.Routes {
		handler := newRouteHandler(route)
		switch route.Method {
		case http.MethodGet:
			s.router.Get(route.Path, handler)
		case http.MethodPost:
			s.router.Post(route.Path, handler)
		case http.MethodPut:
			s.router.Put(route.Path, handler)
		case http.MethodDelete:
			s.router.Delete(route.Path, handler)
		case http.MethodPatch:
			s.router.Patch(route.Path, handler)
		default:
			s.router.Method(route.Method, route.Path, http.HandlerFunc(handler))
		}
	}

	// Catch-all for unmatched routes: return 404 with server signature
	s.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
}
