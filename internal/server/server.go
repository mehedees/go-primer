// Package server assembles the router (routes + middleware) and owns the
// *http.Server lifecycle: start, then shut down gracefully.
//
// Python analogy: this is your `create_app()` factory plus what gunicorn/
// uvicorn do for you (listening, timeouts, draining on SIGTERM). Go's
// net/http is a production-grade server, so there is no separate app server.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	"github.com/mehedees/go-primer/internal/config"
	"github.com/mehedees/go-primer/internal/greeting"
	"github.com/mehedees/go-primer/internal/health"
	"github.com/mehedees/go-primer/internal/httpx"
	"github.com/mehedees/go-primer/internal/middleware"
)

// Server bundles the HTTP server with the pieces needed to stop it cleanly.
type Server struct {
	cfg    config.HTTPConfig
	log    zerolog.Logger
	http   *http.Server
	health *health.Handler
}

// Deps lists everything the router needs. Passing a struct instead of many
// positional arguments keeps call sites readable as the app grows.
type Deps struct {
	Config   config.Config
	Logger   zerolog.Logger
	Version  string
	Greeting *greeting.Handler
}

// New builds the router and the underlying http.Server, but does not start it.
func New(d Deps) *Server {
	hh := health.NewHandler(d.Version)

	s := &Server{
		cfg:    d.Config.HTTP,
		log:    d.Logger,
		health: hh,
	}
	s.http = &http.Server{
		Addr:    d.Config.HTTP.Addr,
		Handler: NewRouter(d, hh),
		// Without timeouts, slow or malicious clients can hold connections
		// open forever. Never ship an http.Server without them.
		ReadHeaderTimeout: d.Config.HTTP.ReadTimeout,
		ReadTimeout:       d.Config.HTTP.ReadTimeout,
		WriteTimeout:      d.Config.HTTP.WriteTimeout,
		IdleTimeout:       d.Config.HTTP.IdleTimeout,
	}
	return s
}

// NewRouter wires middleware and routes. Exported so tests can exercise the
// full stack via httptest without starting a real server.
//
// Middleware order matters. Requests pass through top to bottom, responses
// return bottom to top (an onion):
//
//	RequestID -> RealIP -> RequestLogger -> Recoverer -> Timeout -> handler
//
// RequestLogger sits outside Recoverer so that it still logs the 500 that
// Recoverer writes after a panic.
func NewRouter(d Deps, hh *health.Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)                             // generate/propagate X-Request-Id
	r.Use(chimw.RealIP)                                // trust X-Forwarded-For (only behind a proxy you control!)
	r.Use(middleware.RequestLogger(d.Logger))          // our zerolog access log + ctx logger
	r.Use(middleware.Recoverer)                        // our panic -> 500 JSON
	r.Use(chimw.Timeout(d.Config.HTTP.RequestTimeout)) // cancel ctx after N seconds
	r.Use(chimw.CleanPath)                             // collapse //double//slashes

	// Probes: keep them outside /api so auth middleware added later never
	// blocks them.
	r.Get("/healthz", hh.Live)
	r.Get("/readyz", hh.Ready)

	// Versioned API group. r.Route creates a sub-router with its own prefix;
	// middleware added inside applies only to these routes.
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(chimw.AllowContentType("application/json"))
		r.Get("/hello", d.Greeting.Hello)
		r.Mount("/greetings", d.Greeting.Routes())

		// A deliberately broken route so you can watch Recoverer at work in
		// development. Try: curl localhost:8080/api/v1/panic
		if d.Config.IsDevelopment() {
			r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
				var m map[string]int
				m["boom"] = 1 // writing to a nil map panics
			})
		}
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(w, r, http.StatusNotFound, "route not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(w, r, http.StatusMethodNotAllowed, "method not allowed")
	})

	return r
}

// Run starts the server and blocks until ctx is cancelled (e.g. by SIGTERM)
// or the listener fails. It then shuts down gracefully: stops accepting new
// connections and waits for in-flight requests, up to ShutdownTimeout.
func (s *Server) Run(ctx context.Context) error {
	// A buffered channel of size 1 lets the goroutine send its result and
	// exit even if nobody is receiving yet.
	errCh := make(chan error, 1)

	go func() {
		s.log.Info().Str("addr", s.http.Addr).Msg("http server listening")
		// ListenAndServe blocks. It always returns a non-nil error;
		// ErrServerClosed is the "normal" one after Shutdown.
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	// select waits on several channels at once and runs whichever is ready
	// first. A little like asyncio.wait(..., return_when=FIRST_COMPLETED).
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("listen: %w", err)
		}
		return nil
	case <-ctx.Done():
		s.log.Info().Msg("shutdown signal received")
	}

	// 1. Fail readiness so load balancers stop sending new traffic.
	s.health.SetReady(false)

	// 2. Give in-flight requests time to finish. We use a fresh context
	//    because ctx is already cancelled.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()

	start := time.Now()
	if err := s.http.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	s.log.Info().Dur("took", time.Since(start)).Msg("http server stopped")
	return nil
}
