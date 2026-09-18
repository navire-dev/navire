package runtime

import (
	"testing"
	"time"

	"github.com/navire-dev/navire/dispatcher/internal/config"
	"github.com/navire-dev/navire/shared/heartbeat"
	"github.com/navire-dev/navire/shared/logger"
)

func TestRememberHeartbeatMarksCoreUnavailableAfterTimeout(t *testing.T) {
	app := &App{
		logger:            logger.New("error", true),
		pendingHeartbeats: map[string]time.Time{"old-heartbeat": time.Now().Add(-3 * heartbeatInterval)},
		coreAvailable:     true,
	}

	app.rememberHeartbeat("new-heartbeat", time.Now())

	if app.coreAvailable {
		t.Fatal("rememberHeartbeat() left Core available after heartbeat timeout")
	}
	if _, ok := app.pendingHeartbeats["old-heartbeat"]; ok {
		t.Fatal("rememberHeartbeat() kept the expired heartbeat")
	}
}

func TestForgetHeartbeatMarksCoreUnavailable(t *testing.T) {
	app := &App{
		logger:            logger.New("error", true),
		pendingHeartbeats: map[string]time.Time{"heartbeat-1": time.Now()},
		coreAvailable:     true,
	}

	app.forgetHeartbeat("heartbeat-1")

	if app.coreAvailable {
		t.Fatal("forgetHeartbeat() left Core available after publish failure")
	}
	if len(app.pendingHeartbeats) != 0 {
		t.Fatal("forgetHeartbeat() kept a failed heartbeat")
	}
}

func TestRejectedHeartbeatInvalidatesRegistration(t *testing.T) {
	app := &App{
		cfg:               &config.Config{ConsumerID: "dspc-1"},
		logger:            logger.New("error", true),
		registrationToken: "old-token",
		registrationWake:  make(chan struct{}, 1),
		pendingHeartbeats: map[string]time.Time{"heartbeat-1": time.Now()},
		coreAvailable:     true,
	}

	err := app.processHeartbeatAck(t.Context(), heartbeat.Ack{
		ID:           "core-1:heartbeat-ack:1",
		HeartbeatID:  "heartbeat-1",
		DispatcherID: "dspc-1",
		CoreID:       "core-1",
		OccurredAt:   time.Now().UTC(),
		Accepted:     false,
		Reason:       "dispatcher registration is not valid",
	})
	if err != nil {
		t.Fatalf("processHeartbeatAck() error = %v", err)
	}
	if got := app.currentRegistrationToken(); got != "" {
		t.Fatalf("registration token = %q, want empty", got)
	}
	select {
	case <-app.registrationWake:
	default:
		t.Fatal("rejected heartbeat did not wake registration")
	}
}
