package runtime

import (
	"context"
	"time"

	"github.com/navire-dev/navire/shared/logger"
	"github.com/navire-dev/navire/shared/models"
	"github.com/navire-dev/navire/shared/reporting"
)

func (a *App) processPlan(ctx context.Context, plan models.ExecutionPlan) error {
	a.logger.Info("execution plan received",
		logger.String("message_id", plan.MessageID),
		logger.String("target", plan.Target.Name),
		logger.String("provider", string(plan.Target.Provider)),
	)
	result := a.sender.Execute(ctx, plan)
	a.logger.Info("dispatching target",
		logger.String("message_id", plan.MessageID),
		logger.String("target", result.TargetName),
		logger.String("provider", string(result.Provider)),
		logger.Duration("duration", result.Duration),
	)
	state := reporting.DeliverySuccess
	if result.Err != nil {
		state = reporting.DeliveryFailure
		a.logger.Errorf("delivery failed: %v", result.Err)
	}
	event := reporting.Event{
		ID:                plan.TransportMessageID + ":delivery-result",
		Kind:              reporting.KindDeliveryResult,
		OccurredAt:        time.Now().UTC(),
		DispatcherID:      a.cfg.ConsumerID,
		RegistrationToken: a.currentRegistrationToken(),
		MessageID:         plan.MessageID,
		TargetName:        result.TargetName,
		Provider:          result.Provider,
		State:             state,
		DurationMS:        result.Duration.Milliseconds(),
	}
	if err := a.reportingPublisher.Publish(ctx, event); err != nil {
		a.logger.Errorf("publish delivery reporting: %v", err)
	}
	return nil
}
