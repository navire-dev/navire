package provider

import (
	"testing"

	"github.com/navire-dev/navire/shared/models"
	sharedproviders "github.com/navire-dev/navire/shared/providers"
)

type testProvider struct{}

func (testProvider) PrepareURL(models.Endpoint) (string, error) {
	return "https://example.com", nil
}

func TestRegisterInitializesZeroValueRegistry(t *testing.T) {
	var registry Registry

	if err := registry.Register(sharedproviders.Gotify, testProvider{}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if _, err := registry.Resolve(sharedproviders.Gotify); err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
}
