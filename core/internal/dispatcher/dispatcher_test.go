package dispatcher

import (
	"net/http"
	"testing"

	"github.com/navire-dev/navire/core/internal/dispatcher/sinks"
	"github.com/navire-dev/navire/shared/models"
)

func TestNewRejectsMissingDependencies(t *testing.T) {
	registry, err := sinks.NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error = %v", err)
	}
	tests := []struct {
		name       string
		httpClient *http.Client
		key        []byte
		registry   *sinks.Registry
	}{
		{name: "HTTP client", httpClient: nil, key: []byte("key"), registry: registry},
		{name: "plan key", httpClient: &http.Client{}, key: nil, registry: registry},
		{name: "provider registry", httpClient: &http.Client{}, key: []byte("key"), registry: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := New(test.httpClient, test.key, test.registry, nil); err == nil {
				t.Fatal("New() accepted missing dependency")
			}
		})
	}
}

func TestDispatchRejectsNilService(t *testing.T) {
	var service *Service
	if err := service.Dispatch(t.Context(), models.ExecutionPlan{}); err == nil {
		t.Fatal("Dispatch() accepted a nil service")
	}
}
