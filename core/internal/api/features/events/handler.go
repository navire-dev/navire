package events

import (
	"context"

	"github.com/navire-dev/navire/core/internal/api/deps"
	"github.com/navire-dev/navire/shared/logger"
)

// EventHandler is the HTTP adapter for event ingestion.
type EventHandler struct {
	service *Service
	log     logger.Logger
}

func NewEventHandler(d deps.Deps) *EventHandler {
	return &EventHandler{service: NewService(d), log: d.Logger}
}

func (h *EventHandler) Execute(ctx context.Context, in EventRequest) (any, error) {
	response, err := h.service.Execute(ctx, in)
	if err != nil && h.log != nil {
		h.log.Error("event request failed", logger.Error(err))
	}
	return response, err
}
