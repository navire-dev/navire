package slack

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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestSendWebhookMessage(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "https://hooks.slack.com/services/T000/B000/secret" {
			t.Errorf("URL = %q", request.URL.String())
		}
		if request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q", request.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(request.Body)
		var got payload
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if got.Text != "Build succeeded\n*Service:* navire" {
			t.Errorf("fallback text = %q", got.Text)
		}
		if len(got.Blocks) != 2 {
			t.Fatalf("blocks = %d, want 2", len(got.Blocks))
		}
		if got.Blocks[0].Text.Text != "Build succeeded" {
			t.Errorf("header = %q", got.Blocks[0].Text.Text)
		}
		if got.Blocks[1].Text.Text != "*Service:* navire" {
			t.Errorf("body = %q", got.Blocks[1].Text.Text)
		}
		return response(http.StatusNoContent, ""), nil
	})}

	err := Send(t.Context(), client, models.Endpoint{URL: "https://hooks.slack.com/services/T000/B000/secret"}, messageTemplate.Rendered{
		Title: "Build succeeded", Body: "**Service:** navire",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
}

func TestSendRejectsInvalidWebhookURL(t *testing.T) {
	err := Send(t.Context(), http.DefaultClient, models.Endpoint{URL: "http://example.com/webhook"}, messageTemplate.Rendered{Body: "message"})
	if err == nil {
		t.Fatal("Send() accepted an invalid Slack webhook URL")
	}
}

func TestSendReturnsSlackError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusForbidden, "invalid_token"), nil
	})}
	err := Send(t.Context(), client, models.Endpoint{URL: "https://hooks.slack.com/services/T000/B000/secret"}, messageTemplate.Rendered{Body: "message"})
	if err == nil || !strings.Contains(err.Error(), "slack returned 403 Forbidden: invalid_token") {
		t.Fatalf("Send() error = %v", err)
	}
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
