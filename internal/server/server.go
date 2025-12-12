package server

import (
	"context"
	"net/http"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/config"
)

// Server wraps an http.Server with graceful shutdown
type Server struct {
	httpServer *http.Server
}

// New creates a new server instance
func New(cfg config.Config, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.Addr(),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(shutdownCtx)
}
