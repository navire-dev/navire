package transport

import (
	"context"

	"github.com/navire-dev/navire/shared/models"
)

// ExecutionPlanPublisher publishes immutable notification plans without
// exposing the underlying transport to the caller.
type ExecutionPlanPublisher interface {
	PublishMany(context.Context, []models.ExecutionPlan) ([]MessageID, error)
}
