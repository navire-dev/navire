package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/navire-dev/navire/core/internal/api/deps"
	"github.com/navire-dev/navire/core/internal/api/features/events"
	"github.com/navire-dev/navire/core/internal/api/features/ops/healthz"
	"github.com/navire-dev/navire/core/internal/api/features/ops/readyz"
	registerdispatcher "github.com/navire-dev/navire/core/internal/api/features/register_dispatcher"
)

// RegisterRoutes explicitly assembles the API surface.
// Global middleware belongs to the parent router in api.New. Route-specific
// middleware is passed to Register for the individual feature.
func RegisterRoutes(r chi.Router, d deps.Deps, eventMiddleware ...Middleware) {
	r.Route("/v1", func(v1 chi.Router) {
		Register(v1, d, events.Registrar(), eventMiddleware...)
	})
	Register(r, d, healthz.Registrar())
	Register(r, d, readyz.Registrar())
	Register(r, d, registerdispatcher.Registrar())
}
