// Package dispatcher delivers execution plans to their configured providers.
package dispatcher

import (
	"context"
	"fmt"
	"net/http"

	"github.com/navire-dev/navire/core/internal/dispatcher/sender"
	"github.com/navire-dev/navire/core/internal/dispatcher/sinks"
	"github.com/navire-dev/navire/shared/logger"
	"github.com/navire-dev/navire/shared/models"
)

// Service is the Core-owned dispatcher boundary. It knows how to deliver an
// execution plan, but not how plans are transported or how Core is run.
type Service struct {
	sender sender.Service
}

// New builds a dispatcher service from its delivery dependencies.
func New(httpClient *http.Client, executionPlanKey []byte, registry *sinks.Registry, log logger.Logger) (*Service, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("dispatcher HTTP client is nil")
	}
	if len(executionPlanKey) == 0 {
		return nil, fmt.Errorf("dispatcher plan key is required")
	}
	if registry == nil {
		return nil, fmt.Errorf("dispatcher provider registry is nil")
	}
	return &Service{
		sender: sender.Service{
			HTTPClient:       httpClient,
			ExecutionPlanKey: executionPlanKey,
			Registry:         registry,
			Logger:           log,
		},
	}, nil
}

// Dispatch delivers one immutable execution plan using its target policy.
func (s *Service) Dispatch(ctx context.Context, plan models.ExecutionPlan) error {
	if s == nil {
		return fmt.Errorf("dispatcher service is nil")
	}
	return s.sender.Execute(ctx, plan).Error()
}
