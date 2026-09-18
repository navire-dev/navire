package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/navire-dev/navire/core/internal/api/deps"
)

func TestRegisterRoutesVersionsEventIngestion(t *testing.T) {
	r := chi.NewRouter()
	api := chi.NewRouter()
	r.Mount("/api", api)
	RegisterRoutes(api, deps.Deps{})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(`{"key":"missing","variant":"ok","data":{}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatalf("versioned event route returned status %d", recorder.Code)
	}

	legacy := httptest.NewRequest(http.MethodPost, "/api/notify", strings.NewReader(`{}`))
	legacy.Header.Set("Content-Type", "application/json")
	legacyRecorder := httptest.NewRecorder()
	r.ServeHTTP(legacyRecorder, legacy)

	if legacyRecorder.Code != http.StatusNotFound {
		t.Fatalf("legacy notify route status = %d, want %d", legacyRecorder.Code, http.StatusNotFound)
	}
}
