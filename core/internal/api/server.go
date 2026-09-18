package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/navire-dev/navire/core/internal/api/deps"
	opsmetrics "github.com/navire-dev/navire/core/internal/api/features/ops/metrics"
	"github.com/navire-dev/navire/core/internal/api/mw/global"
	"github.com/navire-dev/navire/core/internal/config"

	"github.com/navire-dev/navire/core/internal/api/router"
	"github.com/navire-dev/navire/core/internal/utils"
	"github.com/navire-dev/navire/shared/logger"
)

// Server wraps the HTTP server and its dependencies.
type Server struct {
	http   *http.Server
	logger logger.Logger
}

// New builds the HTTP server (router, middlewares, route registration).
func New(cfg *config.Config, log logger.Logger, d deps.Deps) *Server {
	r := chi.NewRouter()

	// --- Global middlewares (safe defaults)
	r.Use(middleware.GetHead)
	r.Use(middleware.RequestID) // Adds X-Request-ID / ctx key
	trustedProxies := utils.NewIPMatcher(cfg.AllowedProxies)
	r.Use(global.Log(log, cfg.TrustProxy, trustedProxies))
	r.Use(global.CORS()) // Optional: tune for prod
	r.Use(global.AllowOnlyCIDRS(cfg.AllowedCIDRS, cfg.TrustProxy, trustedProxies))
	r.Use(middleware.Recoverer)                                                                          // Prevent panics from crashing the process
	r.Use(middleware.Compress(5, "application/json", "text/html", "text/css", "application/javascript")) // Gzip responses
	// Enable per-request timeout only if you genuinely need a global cap.
	// Consider removing if you have streaming or SSE endpoints.
	// r.Use(middleware.Timeout(10 * time.Second))

	// API subrouter
	opsmetrics.Registrar()(r, d)
	api := chi.NewRouter()
	r.Mount("/api", api)

	var eventMiddleware []router.Middleware
	if cfg.EventsRateLimit != nil {
		eventMiddleware = append(eventMiddleware, global.RateLimit(global.RateLimitConfig{
			Burst:             cfg.EventsRateLimit.Burst,
			RefillPerIPPerMin: cfg.EventsRateLimit.RefillPerIPPerMin,
			MaxEntries:        cfg.EventsRateLimit.MaxEntries,
			SweepInterval:     cfg.EventsRateLimit.SweepInterval,
			IdleTTL:           cfg.EventsRateLimit.IdleTTL,
			TrustProxy:        cfg.TrustProxy,
			TrustedProxies:    trustedProxies,
		}))
	}
	router.RegisterRoutes(api, d, eventMiddleware...)

	s := &http.Server{
		Addr:              config.InternalListenAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MiB
	}

	s.RegisterOnShutdown(func() {
		uptime := time.Since(d.StartTime).Round(time.Millisecond)
		log.Infof("HTTP server shutdown complete (uptime=%s)", uptime)
	})

	return &Server{
		http:   s,
		logger: log,
	}
}

// Start runs the HTTP server (blocks until error or shutdown).
func (s *Server) Start() error {
	s.logger.Infof("HTTP server listening on %s", s.http.Addr)

	// If TLSConfig is provided upstream, prefer TLS serve.
	var err error
	if s.http.TLSConfig != nil {
		err = s.http.ListenAndServeTLS("", "")
	} else {
		err = s.http.ListenAndServe()
	}

	// http.ErrServerClosed is expected on graceful shutdown.
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Stop gracefully shuts down the server with the provided context deadline.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("HTTP server shutting down...")
	// Disable keep-alives to speed up shutdown drain.
	s.http.SetKeepAlivesEnabled(false)
	return s.http.Shutdown(ctx)
}
