package readyz

import (
	"github.com/go-chi/chi/v5"

	"github.com/navire-dev/navire/core/internal/api/deps"
)

func Registrar() func(r chi.Router, d deps.Deps) {
	return func(r chi.Router, d deps.Deps) {
		r.Get("/readyz", Readyz())
	}
}
