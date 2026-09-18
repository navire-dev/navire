// Package provider contains Core-side provider preparation logic.
// It never sends notifications; it prepares provider-specific delivery data.
package provider

import (
	"fmt"

	"github.com/navire-dev/navire/core/internal/provider/discord"
	"github.com/navire-dev/navire/core/internal/provider/gotify"
	"github.com/navire-dev/navire/core/internal/provider/ntfy"
	"github.com/navire-dev/navire/core/internal/provider/slack"
	"github.com/navire-dev/navire/shared/models"
	sharedproviders "github.com/navire-dev/navire/shared/providers"
)

type Provider interface {
	PrepareURL(models.Endpoint) (string, error)
}

type Registry struct {
	providers map[sharedproviders.ID]Provider
}

func NewDefaultRegistry() (*Registry, error) {
	registry := &Registry{providers: make(map[sharedproviders.ID]Provider)}
	implementations := map[sharedproviders.ID]Provider{
		sharedproviders.Gotify:  gotify.Provider{},
		sharedproviders.Discord: discord.Provider{},
		sharedproviders.Ntfy:    ntfy.Provider{},
		sharedproviders.Slack:   slack.Provider{},
	}
	for _, id := range sharedproviders.ActiveIDs() {
		implementation, ok := implementations[id]
		if !ok {
			return nil, fmt.Errorf("active provider %q has no Core implementation", id)
		}
		if err := registry.Register(id, implementation); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// Register adds a provider during bootstrap. The registry is read-only after
// bootstrap and must not be mutated concurrently with Resolve or PrepareURL.
func (r *Registry) Register(id sharedproviders.ID, provider Provider) error {
	if r == nil {
		return fmt.Errorf("provider registry is nil")
	}
	if id == "" {
		return fmt.Errorf("provider ID is required")
	}
	if provider == nil {
		return fmt.Errorf("provider %q is nil", id)
	}
	if r.providers == nil {
		r.providers = make(map[sharedproviders.ID]Provider)
	}
	if _, exists := r.providers[id]; exists {
		return fmt.Errorf("provider %q is already registered", id)
	}
	r.providers[id] = provider
	return nil
}

func (r *Registry) Resolve(id sharedproviders.ID) (Provider, error) {
	if r == nil {
		return nil, fmt.Errorf("provider registry is nil")
	}
	provider, ok := r.providers[id]
	if !ok {
		return nil, fmt.Errorf("provider %q is not registered in Core", id)
	}
	return provider, nil
}

func (r *Registry) PrepareURL(endpoint models.Endpoint) (string, error) {
	provider, err := r.Resolve(endpoint.Provider)
	if err != nil {
		return "", err
	}
	return provider.PrepareURL(endpoint)
}
