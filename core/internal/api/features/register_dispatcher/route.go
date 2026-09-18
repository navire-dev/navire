package register_dispatcher

import (
	"github.com/go-chi/chi/v5"
	"github.com/navire-dev/navire/core/internal/api/bind"
	"github.com/navire-dev/navire/core/internal/api/deps"
)

func Registrar() func(r chi.Router, d deps.Deps) {
	return func(r chi.Router, d deps.Deps) {
		h := NewHandler(d)
		r.Post("/register-dispatcher", bind.JSONAction(bind.Options{
			MaxBytes:              16 << 10,
			DisallowUnknownFields: true,
			RequireContentType:    "application/json",
			SuccessStatus:         200,
		}, nil, Validate(), h.Execute))
	}
}
