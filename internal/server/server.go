// Package server provides a minimal HTTP server that dispatches to handlers
// with access to the shared store.Store.
package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/alexbathome/albatross/pkg/store"
)

// Server is a minimal HTTP server dispatching to handlers that need access
// to the shared store.Store.
type Server struct {
	mux        *http.ServeMux
	store      store.Store
	httpServer *http.Server
}

// ServerHandlerFunc handles a single route, with s giving access to the
// Server (and its store) that dispatched the request.
type ServerHandlerFunc func(s *Server, w http.ResponseWriter, r *http.Request)

// NewServer returns a Server backed by store.
func NewServer(store store.Store) *Server {
	mux := http.NewServeMux()
	return &Server{
		store: store,
		mux:   mux,
		httpServer: &http.Server{
			Handler: mux,
		},
	}
}

// Register routes pattern (a net/http ServeMux pattern, e.g. "GET /api/foo")
// to handler.
func (s *Server) Register(pattern string, handler ServerHandlerFunc) {
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		handler(s, w, r)
	})
}

// Handle registers a standard http.Handler for pattern, for routes (like a
// swagger UI) that don't need access to the Server itself.
func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

// Store returns the store.Store backing this Server, for use by
// ServerHandlerFuncs that need to read or write persisted data.
func (s *Server) Store() store.Store {
	return s.store
}

// ListenAndServe starts serving on addr, blocking until the server stops or
// fails. It returns nil if stopped via Shutdown.
func (s *Server) ListenAndServe(addr string) error {
	s.httpServer.Addr = addr
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the server, waiting for in-flight requests to
// finish or ctx to be done, whichever comes first.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
