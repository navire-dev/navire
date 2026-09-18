package gotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	messageTemplate "github.com/navire-dev/navire/dispatcher/internal/render"
	"github.com/navire-dev/navire/shared/models"
)

func Send(ctx context.Context, client *http.Client, endpoint models.Endpoint, message messageTemplate.Rendered) error {
	prepared, err := Prepare(endpoint, message)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, prepared.Method, prepared.URL, bytes.NewReader(prepared.Body))
	if err != nil {
		return err
	}
	req.Header = prepared.Headers.Clone()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gotify returned %s", resp.Status)
	}
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestFormatBodySeparatesMarkdownLines(t *testing.T) {
	t.Parallel()

	got := formatBody("**Workspace:** production-platform\r\n\n**Environment:** production\n**Stack:** production-eu\n")
	want := "**Workspace:** production-platform\n\n**Environment:** production\n\n**Stack:** production-eu"
	if got != want {
		t.Fatalf("formatBody() = %q, want %q", got, want)
	}
}

func TestSendAuthenticationModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		auth       models.Auth
		assertAuth func(*testing.T, *http.Request)
	}{
		{
			name: "query token",
			auth: models.Auth{Type: models.AuthQuery, Param: "token", Value: "test-token"},
			assertAuth: func(t *testing.T, request *http.Request) {
				t.Helper()
				if got := request.URL.Query().Get("token"); got != "test-token" {
					t.Errorf("token = %q", got)
				}
			},
		},
		{
			name: "header token",
			auth: models.Auth{Type: models.AuthHeader, Param: "X-Gotify-Key", Value: "test-token"},
			assertAuth: func(t *testing.T, request *http.Request) {
				t.Helper()
				if got := request.Header.Get("X-Gotify-Key"); got != "test-token" {
					t.Errorf("X-Gotify-Key = %q", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				tt.assertAuth(t, request)
				assertPayload(t, request)
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"id":1}`)),
					Request:    request,
				}, nil
			})}

			endpoint := models.Endpoint{
				URL:  "https://gotify.example/message",
				Auth: tt.auth,
			}
			message := messageTemplate.Rendered{
				Title:    "Build succeeded",
				Body:     "Done after 20s.",
				Priority: messageTemplate.PriorityHigh,
			}
			if err := Send(t.Context(), client, endpoint, message); err != nil {
				t.Fatalf("Send() error = %v", err)
			}
		})
	}
}

func assertPayload(t *testing.T, request *http.Request) {
	t.Helper()
	if request.Method != http.MethodPost {
		t.Errorf("method = %s", request.Method)
	}
	if request.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", request.Header.Get("Content-Type"))
	}
	var payload struct {
		Title    string                    `json:"title"`
		Message  string                    `json:"message"`
		Priority int                       `json:"priority"`
		Extras   map[string]map[string]any `json:"extras"`
	}
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Title != "Build succeeded" || payload.Message != "Done after 20s." {
		t.Errorf("payload = %#v", payload)
	}
	if payload.Priority != 8 {
		t.Errorf("priority = %d", payload.Priority)
	}
	if got := payload.Extras["client::display"]["contentType"]; got != "text/markdown" {
		t.Errorf("contentType = %v", got)
	}
}
