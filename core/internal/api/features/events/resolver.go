package events

import (
	"context"
	"fmt"

	"github.com/navire-dev/navire/core/internal/ingest"
	sharedcrypto "github.com/navire-dev/navire/shared/crypto"
	"github.com/navire-dev/navire/shared/models"
)

func (s *Service) resolveEndpoint(ctx context.Context, name string) (ingest.Endpoint, error) {
	if s.d.EndpointStore == nil {
		return ingest.Endpoint{}, fmt.Errorf("endpoint store is not configured")
	}
	endpoint, err := s.d.EndpointStore.GetEndpoint(ctx, name)
	if err != nil {
		return ingest.Endpoint{}, err
	}
	if endpoint.Auth.Value != "" && endpoint.Auth.Type != models.AuthNone {
		value, err := sharedcrypto.Decrypt(endpoint.Auth.Value, s.d.SecretsEncryptionKey)
		if err != nil {
			return ingest.Endpoint{}, fmt.Errorf("decrypt endpoint %s: %w", name, err)
		}
		endpoint.Auth.Value = value
	}
	return endpoint, nil
}
