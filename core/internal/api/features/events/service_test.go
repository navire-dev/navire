package events

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	apierrors "github.com/navire-dev/navire/core/internal/api/apperrors"
	"github.com/navire-dev/navire/core/internal/api/deps"
	"github.com/navire-dev/navire/shared/models"
	templateCatalog "github.com/navire-dev/navire/shared/template/catalog"
	"github.com/navire-dev/navire/shared/transport"
)

type publisherStub struct{}

func (publisherStub) PublishMany(_ context.Context, plans []models.ExecutionPlan) ([]transport.MessageID, error) {
	return make([]transport.MessageID, len(plans)), nil
}

type countingPublisher struct {
	calls int
}

func (p *countingPublisher) PublishMany(_ context.Context, plans []models.ExecutionPlan) ([]transport.MessageID, error) {
	p.calls++
	return make([]transport.MessageID, len(plans)), nil
}

func TestExecuteRejectsMissingTemplateDataBeforePublishing(t *testing.T) {
	t.Parallel()

	templatesDir := t.TempDir()
	templateYAML := `key: test
variants:
  failed:
    state: failure
    title: "{{ .service }} failed"
    body: "Request {{ .request_id }} requires attention."
providers:
  gotify:
    endpoints:
      test: {}
`
	if err := os.WriteFile(filepath.Join(templatesDir, "test.yml"), []byte(templateYAML), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	catalog := templateCatalog.NewCatalog(templatesDir)
	if issues, err := catalog.Reload(); err != nil || len(issues) != 0 {
		t.Fatalf("Reload() issues=%v error=%v", issues, err)
	}

	service := NewService(deps.Deps{
		Publisher:       publisherStub{},
		TemplateCatalog: catalog,
	})
	_, err := service.Execute(t.Context(), EventRequest{
		TemplateKey: "test",
		Variant:     "failed",
		Data:        map[string]any{"service": "booking-api"},
	})
	if err == nil {
		t.Fatal("Execute() accepted an event with missing template data")
	}

	var publicErr *apierrors.Error
	if !errors.As(err, &publicErr) {
		t.Fatalf("Execute() error = %T, want apperrors.Error", err)
	}
	if publicErr.PublicCode() != "template_data_invalid" {
		t.Fatalf("PublicCode() = %q, want template_data_invalid", publicErr.PublicCode())
	}
	if !strings.Contains(err.Error(), "request_id") {
		t.Fatalf("Execute() error = %q, want missing request_id", err)
	}
}

func TestExecuteRejectsUndeclaredVariantBeforePublishing(t *testing.T) {
	t.Parallel()

	templatesDir := t.TempDir()
	templateYAML := `key: test
variants:
  succeeded:
    state: success
    title: "{{ .service }} succeeded"
    body: "The service completed successfully."
providers:
  gotify:
    endpoints:
      test: {}
`
	if err := os.WriteFile(filepath.Join(templatesDir, "test.yml"), []byte(templateYAML), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	catalog := templateCatalog.NewCatalog(templatesDir)
	if issues, err := catalog.Reload(); err != nil || len(issues) != 0 {
		t.Fatalf("Reload() issues=%v error=%v", issues, err)
	}

	publisher := &countingPublisher{}
	service := NewService(deps.Deps{
		Publisher:       publisher,
		TemplateCatalog: catalog,
	})
	_, err := service.Execute(t.Context(), EventRequest{
		TemplateKey: "test",
		Variant:     "missing",
		Data:        map[string]any{"service": "booking-api"},
	})
	if err == nil {
		t.Fatal("Execute() accepted an undeclared variant")
	}

	var publicErr *apierrors.Error
	if !errors.As(err, &publicErr) {
		t.Fatalf("Execute() error = %T, want apperrors.Error", err)
	}
	if publicErr.PublicCode() != "variant_not_found" {
		t.Fatalf("PublicCode() = %q, want variant_not_found", publicErr.PublicCode())
	}
	if publisher.calls != 0 {
		t.Fatalf("PublishMany() calls = %d, want 0", publisher.calls)
	}
}
