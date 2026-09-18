package discord

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
		return fmt.Errorf("discord returned %s", resp.Status)
	}
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestSendEmbed(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Query().Get("wait") != "true" {
			t.Errorf("wait = %q", request.URL.Query().Get("wait"))
		}
		if request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q", request.Header.Get("Content-Type"))
		}
		var payload struct {
			Embeds []struct {
				Title       string `json:"title"`
				Description string `json:"description"`
			} `json:"embeds"`
			AllowedMentions struct {
				Parse []string `json:"parse"`
			} `json:"allowed_mentions"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if len(payload.Embeds) != 1 || payload.Embeds[0].Title != "Build failed" || payload.Embeds[0].Description != "**Service** is down" {
			t.Errorf("embeds = %#v", payload.Embeds)
		}
		if len(payload.AllowedMentions.Parse) != 0 {
			t.Errorf("allowed mentions = %#v", payload.AllowedMentions.Parse)
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Status:     "204 No Content",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    request,
		}, nil
	})}

	err := Send(t.Context(), client, models.Endpoint{URL: "https://discord.com/api/webhooks/id/token"}, messageTemplate.Rendered{
		Title: "Build failed", Body: "**Service** is down",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
}

func TestSendRejectsOversizedEmbed(t *testing.T) {
	message := messageTemplate.Rendered{Title: strings.Repeat("x", 257), Body: "body"}
	if err := Send(t.Context(), &http.Client{}, models.Endpoint{URL: "https://discord.com/webhook"}, message); err == nil {
		t.Fatal("Send() accepted oversized title")
	}
}
