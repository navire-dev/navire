package ntfy

import (
	"bytes"
	"context"
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
		return fmt.Errorf("ntfy returned %s", resp.Status)
	}
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestSendTokenAuthenticatedMarkdownMessage(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "https://ntfy.example.com/mytopic?auth=QmVhcmVyIHRva2Vu" {
			t.Errorf("URL = %q", request.URL.String())
		}
		if request.Header.Get("Content-Type") != "text/markdown" {
			t.Errorf("Content-Type = %q", request.Header.Get("Content-Type"))
		}
		if request.Header.Get("Title") != "Build failed" {
			t.Errorf("Title = %q", request.Header.Get("Title"))
		}
		if request.Header.Get("Priority") != "high" {
			t.Errorf("Priority = %q", request.Header.Get("Priority"))
		}
		body, _ := io.ReadAll(request.Body)
		if string(body) != "**Service** is down" {
			t.Errorf("body = %q", body)
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Status:     "204 No Content",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    request,
		}, nil
	})}

	err := Send(t.Context(), client, models.Endpoint{URL: "https://ntfy.example.com/mytopic?auth=QmVhcmVyIHRva2Vu"}, messageTemplate.Rendered{
		Title: "Build failed", Body: "**Service** is down", Priority: models.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
}

func TestSendCompactsMarkdownParagraphSeparators(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(request.Body)
		if got, want := string(body), "Status: up\nEndpoint: https://example.com\nMessage"; got != want {
			t.Errorf("body = %q, want %q", got, want)
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Status:     "204 No Content",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    request,
		}, nil
	})}

	err := Send(t.Context(), client, models.Endpoint{URL: "https://ntfy.example.com/topic"}, messageTemplate.Rendered{
		Body: "Status: up\n\nEndpoint: https://example.com\n\nMessage",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
}

func TestSendUnauthenticatedMessage(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "https://ntfy.example.com/mytopic" {
			t.Errorf("URL = %q", request.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Status:     "204 No Content",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    request,
		}, nil
	})}

	if err := Send(t.Context(), client, models.Endpoint{URL: "https://ntfy.example.com/mytopic"}, messageTemplate.Rendered{Body: "message"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
}
