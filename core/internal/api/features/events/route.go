package events

import (
	"github.com/go-chi/chi/v5"
	"github.com/navire-dev/navire/core/internal/api/bind"
	"github.com/navire-dev/navire/core/internal/api/deps"
)

func Validate() bind.ValidateFunc {
	return func(v interface{}) []bind.FieldError {
		return v.(*EventRequest).validate(GlobalLimits)
	}
}

func Registrar() func(r chi.Router, d deps.Deps) {
	return func(r chi.Router, d deps.Deps) {
		h := NewEventHandler(d)
		r.Post("/events", bind.JSONAction(bind.Options{
			MaxBytes:              1 << 20,
			DisallowUnknownFields: true,
			RequireContentType:    "application/json",
			SuccessStatus:         202,
		}, nil, Validate(), h.Execute))
	}
}
