package sender

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/navire-dev/navire/dispatcher/internal/sinks"
	sharedcrypto "github.com/navire-dev/navire/shared/crypto"
	"github.com/navire-dev/navire/shared/models"
	sharedproviders "github.com/navire-dev/navire/shared/providers"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestExecuteUsesDecryptedURLAsUnauthenticatedEndpoint(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	encryptedURL, err := sharedcrypto.Encrypt("https://gotify.example/message?token=secret", key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	var gotURL string
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		gotURL = request.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"id":1}`)),
			Request:    request,
		}, nil
	})}

	plan := models.ExecutionPlan{
		MessageID: "message-1",
		Variant: models.ExecutionVariant{
			Name: "default", Title: "Title", Body: "Body", Priority: models.PriorityNormal,
		},
	}
	providerRegistry, err := sinks.NewDefault()
	if err != nil {
		t.Fatalf("NewDefaultRegistry() error = %v", err)
	}
	service := Service{HTTPClient: client, DispatchPlanKey: key, Registry: providerRegistry}
	plan.Target = models.ExecutionTarget{Name: "gotify_prod", Provider: sharedproviders.Gotify, URL: encryptedURL}
	result := service.Execute(t.Context(), plan)
	if err := result.Error(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if gotURL != "https://gotify.example/message?token=secret" {
		t.Fatalf("request URL = %q", gotURL)
	}
}

func TestSafeURLForLogOmitsCredentialBearingParts(t *testing.T) {
	got := safeURLForLog("https://token@example.com/hooks/secret?token=value#fragment")
	if got != "https://example.com" {
		t.Fatalf("safeURLForLog() = %q, want https://example.com", got)
	}
}
