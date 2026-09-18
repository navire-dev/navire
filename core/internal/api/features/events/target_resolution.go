package events

import (
	"context"
	"errors"
	"fmt"

	"github.com/navire-dev/navire/core/internal/ingest"
	sharedcrypto "github.com/navire-dev/navire/shared/crypto"
	"github.com/navire-dev/navire/shared/logger"
	"github.com/navire-dev/navire/shared/models"
	messageTemplate "github.com/navire-dev/navire/shared/template"
	"github.com/navire-dev/navire/shared/template/routing"
)

func configuredDestinations(definition *messageTemplate.Definition) []routing.Destination {
	destinations := make([]routing.Destination, 0)
	for _, providerName := range definition.ProviderNames() {
		for _, endpointName := range definition.EndpointNames(providerName) {
			destinations = append(destinations, routing.Destination{
				Provider: providerName,
				Endpoint: endpointName,
			})
		}
	}
	return destinations
}

func (s *Service) resolveTarget(ctx context.Context, definition *messageTemplate.Definition, destination routing.Destination) (models.ExecutionTarget, bool, string, error) {
	fullName := destination.Provider + "_" + destination.Endpoint
	endpoint, err := s.resolveEndpoint(ctx, fullName)
	if errors.Is(err, ingest.ErrEndpointNotFound) {
		return models.ExecutionTarget{}, false, "", nil
	}
	if err != nil {
		return models.ExecutionTarget{}, false, "endpoint_resolution", err
	}
	resolvedURL, err := s.d.ProviderRegistry.PrepareURL(endpoint)
	if err != nil {
		return models.ExecutionTarget{}, false, "target_url", fmt.Errorf("prepare target %s: %w", fullName, err)
	}
	encryptedURL, err := sharedcrypto.Encrypt(resolvedURL, s.d.DispatchPlanKey)
	if err != nil {
		return models.ExecutionTarget{}, false, "target_encryption", fmt.Errorf("encrypt target %s: %w", fullName, err)
	}
	policy, err := definition.ResolvePolicy(destination.Provider, destination.Endpoint)
	if err != nil {
		return models.ExecutionTarget{}, false, "delivery_policy", fmt.Errorf("resolve policy for target %s: %w", fullName, err)
	}
	target := models.ExecutionTarget{
		Name:     endpoint.FullName,
		Provider: endpoint.Provider,
		URL:      encryptedURL,
		Policy:   policy,
	}
	if s.d.Logger != nil {
		s.d.Logger.Info("execution target selected",
			logger.String("target", endpoint.FullName),
			logger.String("provider", string(endpoint.Provider)),
		)
	}
	return target, true, "", nil
}
