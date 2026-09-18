package metrics

import (
	"github.com/go-chi/chi/v5"

	"github.com/navire-dev/navire/core/internal/api/deps"
)

func Registrar() func(chi.Router, deps.Deps) {
	return func(r chi.Router, d deps.Deps) {
		service := NewService(d.MetricsExporter)
		r.Get("/metrics", NewHandler(service).ServeHTTP)
	}
}
