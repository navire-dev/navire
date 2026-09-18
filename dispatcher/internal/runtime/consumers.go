package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/navire-dev/navire/shared/heartbeat"
	"github.com/navire-dev/navire/shared/lifecycle"
	"github.com/navire-dev/navire/shared/logger"
	redisClient "github.com/navire-dev/navire/shared/redis"
	sharedruntime "github.com/navire-dev/navire/shared/runtime"
	"github.com/navire-dev/navire/shared/transport/redisstreams"
)

func (a *App) consumeUntilAvailable(ctx context.Context) {
	a.logger.Info("dispatcher consumer started",
		logger.String("consumer", a.cfg.ConsumerID),
		logger.String("stream", redisstreams.NotificationStream),
		logger.String("group", redisstreams.NotificationGroup),
	)
	delay := redisRetryInitial
	for {
		if err := redisClient.WaitForReady(ctx, a.redis, a.cfg.Redis, a.logger); err != nil {
			return
		}
		err := a.runConsumers(ctx)
		if err == nil || errors.Is(err, context.Canceled) {
			return
		}
		if ctx.Err() != nil {
			return
		}
		a.logger.Warn("Redis consumer unavailable, retrying",
			logger.Duration("retry_in", delay),
			logger.Error(err))
		if !sharedruntime.WaitForRetry(ctx, delay) {
			return
		}
		delay = sharedruntime.NextRetryDelay(delay, redisRetryMax)
	}
}

func (a *App) runConsumers(ctx context.Context) error {
	consumerCtx, cancel := context.WithCancel(ctx)
	errs := make(chan error, 3)
	var consumers sync.WaitGroup
	startConsumer := func(name string, run func() error) {
		consumers.Add(1)
		go func() {
			defer consumers.Done()
			err := run()
			if err == nil && consumerCtx.Err() == nil {
				err = fmt.Errorf("%s stopped unexpectedly", name)
			}
			errs <- err
		}()
	}
	startConsumer("notification consumer", func() error { return a.consumer.Run(consumerCtx, a.processPlan) })
	startConsumer("lifecycle consumer", func() error { return a.lifecycleConsumer.Run(consumerCtx, a.processLifecycle) })
	startConsumer("heartbeat acknowledgement consumer", func() error {
		return a.heartbeatAckConsumer.Run(consumerCtx, a.processHeartbeatAck)
	})

	var result error
	select {
	case <-ctx.Done():
		result = ctx.Err()
	case err := <-errs:
		result = err
	}
	cancel()
	consumers.Wait()
	return result
}

func (a *App) processHeartbeatAck(_ context.Context, ack heartbeat.Ack) error {
	if ack.DispatcherID != a.cfg.ConsumerID {
		return fmt.Errorf("heartbeat acknowledgement targets dispatcher %q", ack.DispatcherID)
	}
	a.heartbeatMu.Lock()
	if _, ok := a.pendingHeartbeats[ack.HeartbeatID]; !ok {
		a.heartbeatMu.Unlock()
		a.logger.Debug("ignored stale Core heartbeat acknowledgement",
			logger.String("core_id", ack.CoreID),
			logger.String("heartbeat_id", ack.HeartbeatID))
		return nil
	}
	delete(a.pendingHeartbeats, ack.HeartbeatID)
	a.coreAvailable = ack.Accepted
	a.heartbeatMu.Unlock()
	if !ack.Accepted {
		a.invalidateRegistration()
	}
	if ack.Accepted {
		a.logger.Debug("Core heartbeat acknowledged",
			logger.String("core_id", ack.CoreID),
			logger.String("heartbeat_id", ack.HeartbeatID))
	} else {
		a.logger.Warn("Core rejected dispatcher heartbeat",
			logger.String("core_id", ack.CoreID),
			logger.String("heartbeat_id", ack.HeartbeatID),
			logger.String("reason", ack.Reason))
	}
	return nil
}

func (a *App) processLifecycle(_ context.Context, event lifecycle.Event) error {
	if !a.startedAt.IsZero() && event.OccurredAt.Before(a.startedAt) {
		a.logger.Debug("ignored stale Core lifecycle event",
			logger.String("kind", string(event.Kind)),
			logger.String("core_id", event.CoreID),
			logger.String("occurred_at", event.OccurredAt.Format(time.RFC3339Nano)),
		)
		return nil
	}
	a.logger.Info("Core lifecycle event received",
		logger.String("kind", string(event.Kind)),
		logger.String("core_id", event.CoreID),
		logger.String("reason", event.Reason),
		logger.String("occurred_at", event.OccurredAt.Format(time.RFC3339Nano)),
	)
	if event.Kind == lifecycle.CoreShutdown {
		a.markCoreUnavailable("Core shutdown event received")
		a.logger.Warn("Core shutdown received and acknowledged",
			logger.String("core_id", event.CoreID),
			logger.String("reason", event.Reason),
		)
	}
	return nil
}
