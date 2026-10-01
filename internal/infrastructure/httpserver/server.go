package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Dependencies contains application services required by HTTP handlers
type Dependencies struct {
	Monitors MonitorService
	Checks   CheckService
	WebDir   string
	Logger   *slog.Logger
}

// Server manages the HTTP server lifecycle
type Server struct {
	httpServer *http.Server
}

// NewServer creates an HTTP server from application dependencies
func NewServer(addr string, dependencies Dependencies) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: NewHandler(dependencies),
		},
	}
}

// NewHandler creates an HTTP handler from application services
func NewHandler(dependencies Dependencies) http.Handler {
	handler := newHandler(dependencies)
	router := chi.NewRouter()
	handler.registerRoutes(router)
	return router
}

// Start runs the HTTP server in a separate goroutine
func (s *Server) Start(logger *slog.Logger, stop func()) {
	if logger == nil {
		logger = slog.Default()
	}
	if stop == nil {
		stop = func() {}
	}

	go func() {
		logger.Info("HTTP server started", "addr", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server failed", "error", err)
			stop()
		}
	}()
}

// Shutdown gracefully stops the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
