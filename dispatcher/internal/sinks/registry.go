package sinks

import (
	"fmt"

	messageTemplate "github.com/navire-dev/navire/dispatcher/internal/render"
	"github.com/navire-dev/navire/dispatcher/internal/sinks/contract"
	"github.com/navire-dev/navire/dispatcher/internal/sinks/providers/discord"
	"github.com/navire-dev/navire/dispatcher/internal/sinks/providers/gotify"
	"github.com/navire-dev/navire/dispatcher/internal/sinks/providers/ntfy"
	"github.com/navire-dev/navire/dispatcher/internal/sinks/providers/slack"
	"github.com/navire-dev/navire/shared/models"
	sharedproviders "github.com/navire-dev/navire/shared/providers"
)

// Provider delivers one already-rendered notification.
type Provider interface {
	Prepare(models.Endpoint, messageTemplate.Rendered) (contract.PreparedRequest, error)
}

type providerFunc func(models.Endpoint, messageTemplate.Rendered) (contract.PreparedRequest, error)

func (fn providerFunc) Prepare(endpoint models.Endpoint, message messageTemplate.Rendered) (contract.PreparedRequest, error) {
	return fn(endpoint, message)
}

type registration struct {
	id       sharedproviders.ID
	provider Provider
}

// Registry resolves provider IDs to isolated sink implementations.
type Registry struct {
	providers map[sharedproviders.ID]Provider
}

func New(registrations ...registration) (*Registry, error) {
	registry := &Registry{providers: make(map[sharedproviders.ID]Provider, len(registrations))}
	for _, item := range registrations {
		if err := registry.Register(item.id, item.provider); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// Register adds a provider during bootstrap. The registry is read-only after
// bootstrap and must not be mutated concurrently with Resolve.
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
		return nil, fmt.Errorf("unsupported provider %q", id)
	}
	return provider, nil
}

func NewDefault() (*Registry, error) {
	implementations := map[sharedproviders.ID]Provider{
		sharedproviders.Gotify:  providerFunc(gotify.Prepare),
		sharedproviders.Discord: providerFunc(discord.Prepare),
		sharedproviders.Ntfy:    providerFunc(ntfy.Prepare),
		sharedproviders.Slack:   providerFunc(slack.Prepare),
	}
	registrations := make([]registration, 0, len(sharedproviders.ActiveIDs()))
	for _, id := range sharedproviders.ActiveIDs() {
		implementation, ok := implementations[id]
		if !ok {
			return nil, fmt.Errorf("active provider %q has no DSPC implementation", id)
		}
		registrations = append(registrations, registration{id: id, provider: implementation})
	}
	return New(registrations...)
}
