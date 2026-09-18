package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/navire-dev/navire/shared/heartbeat"
	"github.com/navire-dev/navire/shared/logger"
)

const heartbeatInterval = 10 * time.Second

func (a *App) publishHeartbeats(ctx context.Context) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			heartbeatID := fmt.Sprintf("%s:heartbeat:%d", a.cfg.ConsumerID, now.UnixNano())
			a.rememberHeartbeat(heartbeatID, now)
			lag, err := a.consumer.Lag(ctx)
			connected := err == nil
			if err != nil {
				a.logger.Warn("read dispatcher consumer lag", logger.Error(err))
			}
			event := heartbeat.Event{
				ID:                heartbeatID,
				OccurredAt:        now.UTC(),
				DispatcherID:      a.cfg.ConsumerID,
				RegistrationToken: a.currentRegistrationToken(),
				Status:            heartbeat.Ready,
				RedisConnected:    connected,
				StreamLag:         lag,
			}
			if err := a.heartbeatPublisher.Publish(ctx, event); err != nil {
				a.forgetHeartbeat(heartbeatID)
				a.logger.Warn("publish dispatcher heartbeat", logger.Error(err))
			}
		}
	}
}

func (a *App) rememberHeartbeat(id string, occurredAt time.Time) {
	a.heartbeatMu.Lock()
	defer a.heartbeatMu.Unlock()
	timedOut := false
	for heartbeatID, sentAt := range a.pendingHeartbeats {
		if occurredAt.Sub(sentAt) > 2*heartbeatInterval {
			delete(a.pendingHeartbeats, heartbeatID)
			timedOut = true
		}
	}
	if timedOut && a.coreAvailable {
		a.coreAvailable = false
		a.logger.Warn("Core heartbeat acknowledgement timed out")
	}
	a.pendingHeartbeats[id] = occurredAt
	if timedOut {
		a.invalidateRegistration()
	}
}

func (a *App) forgetHeartbeat(id string) {
	a.heartbeatMu.Lock()
	delete(a.pendingHeartbeats, id)
	wasAvailable := a.coreAvailable
	a.coreAvailable = false
	a.heartbeatMu.Unlock()
	if wasAvailable {
		a.logger.Warn("Core heartbeat could not be published")
	}
	a.invalidateRegistration()
}

func (a *App) markCoreUnavailable(reason string) {
	a.heartbeatMu.Lock()
	wasAvailable := a.coreAvailable
	a.coreAvailable = false
	a.heartbeatMu.Unlock()
	if wasAvailable {
		a.logger.Warn("Core marked unavailable", logger.String("reason", reason))
	}
	a.invalidateRegistration()
}
