package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cathal/blascet-photo-advisor/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server wraps the HTTP server and dependencies
type Server struct {
	router *chi.Mux
	db     *db.DB
	addr   string
}

// New creates a new HTTP server
func New(database *db.DB, addr string) *Server {
	s := &Server{
		router: chi.NewRouter(),
		db:     database,
		addr:   addr,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(60 * time.Second))
}

func (s *Server) setupRoutes() {
	// Static assets
	staticDir := "./web/static"
	if _, err := os.Stat(staticDir); err == nil {
		s.router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	}

	// Routes
	s.router.Get("/", s.handleDashboard)
	s.router.Post("/upload", s.handleUpload)
	s.router.Get("/jobs/{id}", s.handleJobDetail)
	s.router.Get("/jobs/{id}/events", s.handleJobEvents)
	s.router.Get("/healthz", s.handleHealth)
}

// Start starts the HTTP server with graceful shutdown
func (s *Server) Start() error {
	server := &http.Server{
		Addr:         s.addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for errors from the server
	serverErrors := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		slog.Info("starting HTTP server", "addr", s.addr)
		serverErrors <- server.ListenAndServe()
	}()

	// Channel to listen for interrupt signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or an error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		slog.Info("received shutdown signal", "signal", sig)

		// Give outstanding requests 30 seconds to complete
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			server.Close()
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}

		slog.Info("server stopped gracefully")
		return nil
	}
}
