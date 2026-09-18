package register_dispatcher

import (
	"context"

	"github.com/navire-dev/navire/core/internal/api/deps"
)

type Handler struct {
	service *Service
}

func NewHandler(d deps.Deps) *Handler {
	return &Handler{service: NewService(d)}
}

func (h *Handler) Execute(ctx context.Context, request Request) (any, error) {
	return h.service.Execute(ctx, request)
}
