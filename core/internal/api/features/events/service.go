package events

import (
	"context"
	"fmt"
	"strings"
	"time"

	apierrors "github.com/navire-dev/navire/core/internal/api/apperrors"
	"github.com/navire-dev/navire/core/internal/api/deps"
	"github.com/navire-dev/navire/shared/logger"
	"github.com/navire-dev/navire/shared/models"
	messageTemplate "github.com/navire-dev/navire/shared/template"
	templateCatalog "github.com/navire-dev/navire/shared/template/catalog"
	"github.com/navire-dev/navire/shared/template/routing"
)

type Service struct {
	d   deps.Deps
	now func() time.Time
}

func NewService(d deps.Deps) *Service {
	now := d.TimeNow
	if now == nil {
		now = time.Now
	}
	return &Service{d: d, now: now}
}

func (s *Service) Execute(ctx context.Context, in EventRequest) (response any, err error) {
	if s.d.Metrics != nil {
		s.d.Metrics.EventReceived()
	}
	accepted := false
	rejectionReason := "internal"
	defer func() {
		if !accepted && s.d.Metrics != nil {
			s.d.Metrics.EventRejected(rejectionReason)
		}
	}()
	in.Timestamp = s.now()
	if s.d.Publisher == nil {
		rejectionReason = "publisher_not_configured"
		return nil, apierrors.Unavailable("publisher_unavailable", "event publisher is unavailable", fmt.Errorf("publisher is not configured"))
	}

	definition, err := s.loadDefinition(in.TemplateKey)
	if err != nil {
		rejectionReason = "template_load"
		return nil, apierrors.NotFound("template_not_found", "template not found", err)
	}
	variantName, variant, resolveErr := definition.ResolveVariant(in.Variant, in.Data)
	if resolveErr != nil {
		rejectionReason = "variant_not_found"
		return nil, apierrors.NotFound("variant_not_found", "event variant not found", resolveErr)
	}
	in.Variant = variantName
	in.EnsureIdempotencyKey()
	priority, inputRejection, err := resolveVariantInput(definition, variantName, in.Priority, in.Data)
	if err != nil {
		rejectionReason = inputRejection
		return nil, err
	}

	targets, rejectionReason, err := s.resolveTargets(ctx, definition, routing.Event{
		Priority: priority,
		State:    variant.State,
		Data:     in.Data,
	})
	if err != nil {
		if rejectionReason == "endpoint_resolution" {
			return nil, apierrors.Unavailable("endpoint_resolution_failed", "event endpoints are temporarily unavailable", err)
		}
		return nil, apierrors.Unprocessable("event_configuration_invalid", "event configuration is invalid", err)
	}

	plans := buildExecutionPlans(in, variantName, variant, priority, targets)
	publicationIDs, err := s.d.Publisher.PublishMany(ctx, plans)
	if err != nil {
		rejectionReason = "publish"
		if s.d.Metrics != nil {
			s.d.Metrics.SetRedisConnected(false)
			s.d.Metrics.RedisOperationError("publish")
			s.d.Metrics.EventPublishError()
		}
		return nil, apierrors.Unavailable("publisher_unavailable", "event could not be accepted", err)
	}
	accepted = true
	publicationIDStrings := make([]string, len(publicationIDs))
	for i, id := range publicationIDs {
		publicationIDStrings[i] = string(id)
	}
	if s.d.Metrics != nil {
		s.d.Metrics.EventAccepted()
		s.d.Metrics.Event(in.TemplateKey, variantName, string(variant.State))
		s.d.Metrics.EventPublished()
		s.d.Metrics.StreamPublished(len(plans))
		s.d.Metrics.SetRedisConnected(true)
	}
	if s.d.Logger != nil {
		s.d.Logger.Info("execution plan published",
			logger.String("message_id", in.IdempotencyKey),
			logger.Int("targets", len(plans)),
			logger.String("publication_ids", strings.Join(publicationIDStrings, ",")),
		)
	}
	return map[string]any{
		"status":          "accepted",
		"publication_ids": publicationIDs,
		"idempotency_key": in.IdempotencyKey,
		"message_id":      in.IdempotencyKey,
	}, nil
}

func (s *Service) loadDefinition(key string) (*messageTemplate.Definition, error) {
	if s.d.TemplateCatalog != nil {
		return s.d.TemplateCatalog.LoadByKey(key)
	}
	return templateCatalog.LoadByKey(s.d.TemplatesDir, key)
}

func buildExecutionPlans(in EventRequest, variantName string, variant messageTemplate.Variant, priority messageTemplate.Priority, targets []models.ExecutionTarget) []models.ExecutionPlan {
	plans := make([]models.ExecutionPlan, 0, len(targets))
	for _, target := range targets {
		plans = append(plans, models.ExecutionPlan{
			SchemaVersion:  models.ExecutionSchemaVersion,
			MessageID:      in.IdempotencyKey,
			TenantID:       in.TenantID,
			IdempotencyKey: in.IdempotencyKey + ":" + target.Name,
			CreatedAt:      in.Timestamp,
			Variant: models.ExecutionVariant{
				Name:     variantName,
				Title:    variant.Title,
				Body:     variant.Body,
				Priority: priority,
				State:    variant.State,
			},
			Data:   in.Data,
			Target: target,
		})
	}
	return plans
}

func resolveVariantInput(definition *messageTemplate.Definition, variantName, priorityOverride string, data map[string]any) (messageTemplate.Priority, string, error) {
	priority, err := definition.ResolvePriority(variantName, priorityOverride)
	if err != nil {
		return "", "priority", apierrors.Unprocessable("invalid_priority", "event priority is invalid", err)
	}
	if err := definition.ValidateData(variantName, data); err != nil {
		return "", "template_data", apierrors.Unprocessable("template_data_invalid", "event data cannot render the selected variant", err)
	}
	return priority, "", nil
}

func (s *Service) resolveTargets(ctx context.Context, definition *messageTemplate.Definition, event routing.Event) ([]models.ExecutionTarget, string, error) {
	if s.d.ProviderRegistry == nil {
		return nil, "provider_registry_not_configured", fmt.Errorf("provider registry is not configured")
	}
	selected := routing.Resolve(definition.Rules, event, configuredDestinations(definition))
	targets := make([]models.ExecutionTarget, 0)
	for _, destination := range selected {
		target, found, reason, err := s.resolveTarget(ctx, definition, destination)
		if err != nil {
			return nil, reason, err
		}
		if !found {
			continue
		}
		targets = append(targets, target)
	}
	if len(targets) == 0 {
		return nil, "no_active_targets", fmt.Errorf("template %q has no active targets", definition.Key)
	}
	return targets, "", nil
}
