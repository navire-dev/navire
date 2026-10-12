package sender

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/navire-dev/navire/core/internal/dispatcher/sinks"
	"github.com/navire-dev/navire/core/internal/dispatcher/sinks/contract"
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
	service := Service{HTTPClient: client, ExecutionPlanKey: key, Registry: providerRegistry}
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

func TestRetryAfterParsesSeconds(t *testing.T) {
	delay, ok := retryAfter("12", time.Now())
	if !ok {
		t.Fatal("retryAfter() did not recognize seconds")
	}
	if delay != 12*time.Second {
		t.Fatalf("retryAfter() = %s, want 12s", delay)
	}
}

func TestRetryAfterParsesHTTPDate(t *testing.T) {
	now := time.Date(2026, time.October, 12, 14, 0, 0, 0, time.UTC)
	delay, ok := retryAfter("Mon, 12 Oct 2026 14:00:12 GMT", now)
	if !ok {
		t.Fatal("retryAfter() did not recognize HTTP date")
	}
	if delay != 12*time.Second {
		t.Fatalf("retryAfter() = %s, want 12s", delay)
	}
}

func TestRetryAfterOverridesConfiguredBackoff(t *testing.T) {
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Status:     "429 Too Many Requests",
				Header:     http.Header{"Retry-After": []string{"0"}},
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    request,
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("ok")),
			Request:    request,
		}, nil
	})}

	service := Service{HTTPClient: client}
	prepared := contract.PreparedRequest{
		Provider: "discord",
		Method:   http.MethodPost,
		URL:      "https://example.com/webhook",
	}
	started := time.Now()
	err := service.sendWithPolicy(t.Context(), prepared, models.DeliveryPolicy{
		Timeout: "1s",
		Retry: models.RetryPolicy{
			MaxAttempts: 2,
			Backoff:     models.BackoffExponential,
			InitialWait: "1m",
		},
	})
	if err != nil {
		t.Fatalf("sendWithPolicy() error = %v", err)
	}
	if requests != 2 {
		t.Fatalf("request count = %d, want 2", requests)
	}
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Fatalf("retry took %s; Retry-After did not override the configured backoff", elapsed)
	}
}

func TestSendWithPolicyPreservesHTTPFailureMetadata(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Status:     "429 Too Many Requests",
			Header:     http.Header{"Retry-After": []string{"12"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    request,
		}, nil
	})}

	service := Service{HTTPClient: client}
	err := service.sendWithPolicy(t.Context(), contract.PreparedRequest{
		Provider: "discord",
		Method:   http.MethodPost,
		URL:      "https://example.com/webhook",
	}, models.DeliveryPolicy{
		Timeout: "1s",
		Retry: models.RetryPolicy{
			MaxAttempts: 1,
		},
	})

	var deliveryErr *DeliveryError
	if !errors.As(err, &deliveryErr) {
		t.Fatalf("sendWithPolicy() error type = %T, want *DeliveryError", err)
	}
	if deliveryErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status code = %d, want %d", deliveryErr.StatusCode, http.StatusTooManyRequests)
	}
	if deliveryErr.Status != "429 Too Many Requests" {
		t.Fatalf("status = %q, want 429 Too Many Requests", deliveryErr.Status)
	}
	if !deliveryErr.HasRetryAfter || deliveryErr.RetryAfter != 12*time.Second {
		t.Fatalf("retry-after = %s (present=%t), want 12s", deliveryErr.RetryAfter, deliveryErr.HasRetryAfter)
	}
	if deliveryErr.Attempts != 1 || deliveryErr.MaxAttempts != 1 {
		t.Fatalf("attempts = %d/%d, want 1/1", deliveryErr.Attempts, deliveryErr.MaxAttempts)
	}
}
