package runtime

import (
	"testing"
	"time"

	"github.com/navire-dev/navire/dispatcher/internal/config"
	"github.com/navire-dev/navire/shared/lifecycle"
	"github.com/navire-dev/navire/shared/logger"
)

func TestProcessLifecycleInvalidatesRegistrationOnCoreShutdown(t *testing.T) {
	app := &App{
		cfg:               &config.Config{ConsumerID: "dspc-1"},
		logger:            logger.New("error", true),
		registrationToken: "old-token",
		registrationWake:  make(chan struct{}, 1),
		coreAvailable:     true,
	}

	err := app.processLifecycle(t.Context(), lifecycle.Event{
		ID:         "core-1:shutdown:1",
		Kind:       lifecycle.CoreShutdown,
		OccurredAt: time.Now().UTC(),
		CoreID:     "core-1",
		Reason:     "context_canceled",
	})
	if err != nil {
		t.Fatalf("processLifecycle() error = %v", err)
	}
	if got := app.currentRegistrationToken(); got != "" {
		t.Fatalf("registration token = %q, want empty", got)
	}
	if app.coreAvailable {
		t.Fatal("Core remained available after shutdown")
	}
	select {
	case <-app.registrationWake:
	default:
		t.Fatal("Core shutdown did not wake registration")
	}
}

func TestProcessLifecycleIgnoresEventsBeforeStartup(t *testing.T) {
	app := &App{
		cfg:               &config.Config{ConsumerID: "dspc-1"},
		logger:            logger.New("error", true),
		startedAt:         time.Date(2026, time.August, 28, 9, 0, 0, 0, time.UTC),
		registrationToken: "current-token",
		coreAvailable:     true,
	}

	err := app.processLifecycle(t.Context(), lifecycle.Event{
		ID:         "core-1:shutdown:1",
		Kind:       lifecycle.CoreShutdown,
		OccurredAt: time.Date(2026, time.August, 28, 8, 59, 0, 0, time.UTC),
		CoreID:     "core-1",
		Reason:     "context_canceled",
	})
	if err != nil {
		t.Fatalf("processLifecycle() error = %v", err)
	}
	if got := app.currentRegistrationToken(); got != "current-token" {
		t.Fatalf("registration token = %q, want current-token", got)
	}
	if !app.coreAvailable {
		t.Fatal("Core was marked unavailable by a stale lifecycle event")
	}
}
