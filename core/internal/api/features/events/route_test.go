package events

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/navire-dev/navire/core/internal/api/deps"
	templateCatalog "github.com/navire-dev/navire/shared/template/catalog"
)

func TestRouteReturnsNotFoundForDiscardedTemplate(t *testing.T) {
	router := chi.NewRouter()
	Registrar()(router, deps.Deps{
		Publisher:       publisherStub{},
		TemplateCatalog: templateCatalog.NewCatalog(t.TempDir()),
	})

	request := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(`{"key":"missing","variant":"ok","data":{}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"template_not_found"`) {
		t.Fatalf("response = %s, want template_not_found", recorder.Body.String())
	}
}
