package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navire-dev/navire/core/internal/api/deps"
)

// Registrar is a function that registers HTTP routes.
type Registrar func(r chi.Router, d deps.Deps)

// Middleware is a standard chi middleware.
type Middleware = func(http.Handler) http.Handler

// Register applies a feature registration function with optional route-local
// middleware. Global middleware belongs on the parent chi router via Use.
func Register(r chi.Router, d deps.Deps, register Registrar, middleware ...Middleware) {
	if len(middleware) == 0 {
		register(r, d)
		return
	}

	register(r.With(middleware...), d)
}
