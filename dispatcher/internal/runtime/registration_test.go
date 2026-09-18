package runtime

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/navire-dev/navire/dispatcher/internal/config"
)

func TestWaitForRegistrationRetryCanBeWokenByCoreSignal(t *testing.T) {
	app := &App{registrationWake: make(chan struct{}, 1)}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	result := make(chan bool, 1)
	go func() { result <- app.waitForRegistrationRetry(ctx, time.Hour) }()
	app.requestRegistration()

	if !<-result {
		t.Fatal("waitForRegistrationRetry() returned false after a registration wake signal")
	}
}

func TestRegisterRequestsANewTokenFromCore(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/register-dispatcher" {
			t.Fatalf("registration request = %s %s", r.Method, r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"status":"registered","dispatcher_id":"dspc-1","registration_token":"new-token","lease_seconds":30}`)),
		}, nil
	})}

	token, err := register(t.Context(), &config.Config{CoreURL: "http://core.test", ConsumerID: "dspc-1"}, []byte("registration-secret"), client)
	if err != nil {
		t.Fatalf("register() error = %v", err)
	}
	if token != "new-token" {
		t.Fatalf("register() token = %q, want %q", token, "new-token")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
