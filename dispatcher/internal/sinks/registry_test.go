package sinks

import (
	"testing"

	messageTemplate "github.com/navire-dev/navire/dispatcher/internal/render"
	"github.com/navire-dev/navire/dispatcher/internal/sinks/contract"
	"github.com/navire-dev/navire/shared/models"
)

func TestResolvesRegisteredProvider(t *testing.T) {
	called := false
	provider := providerFunc(func(models.Endpoint, messageTemplate.Rendered) (contract.PreparedRequest, error) {
		called = true
		return contract.PreparedRequest{}, nil
	})
	registry, err := New(registration{id: "test", provider: provider})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	resolved, err := registry.Resolve("test")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if _, err := resolved.Prepare(models.Endpoint{}, messageTemplate.Rendered{}); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if !called {
		t.Fatal("registered provider was not called")
	}
}

func TestRejectsUnknownAndDuplicateProviders(t *testing.T) {
	provider := providerFunc(func(models.Endpoint, messageTemplate.Rendered) (contract.PreparedRequest, error) {
		return contract.PreparedRequest{}, nil
	})
	registry, err := New(registration{id: "test", provider: provider})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := registry.Register("test", provider); err == nil {
		t.Fatal("Register() accepted a duplicate provider")
	}
	if _, err := registry.Resolve("missing"); err == nil {
		t.Fatal("Resolve() accepted an unknown provider")
	}
}

func TestRegisterInitializesZeroValueRegistry(t *testing.T) {
	provider := providerFunc(func(models.Endpoint, messageTemplate.Rendered) (contract.PreparedRequest, error) {
		return contract.PreparedRequest{}, nil
	})
	var registry Registry

	if err := registry.Register("test", provider); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if _, err := registry.Resolve("test"); err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
}
