package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/navire-dev/navire/shared/heartbeat"
	"github.com/navire-dev/navire/shared/logger"
	"github.com/navire-dev/navire/shared/reporting"
	templateCatalog "github.com/navire-dev/navire/shared/template/catalog"
	"github.com/navire-dev/navire/shared/transport/redisstreams"
)

func (a *App) watchConfig(ctx context.Context) error {
	a.logger.Infof("📁 Watching %s for config changes...", a.cfg.TargetsDir)
	if err := a.ingester.Watch(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("config watcher stopped: %w", err)
	}
	return nil
}

func (a *App) watchTemplates(ctx context.Context) error {
	a.logger.Infof("📁 Watching %s for template changes...", a.cfg.TemplatesDir)
	err := a.templateCatalog.Watch(ctx, func(issues []templateCatalog.Issue, err error) {
		if err != nil {
			a.logger.Errorf("Template watcher reload failed: %v", err)
			return
		}
		for _, issue := range issues {
			a.logger.Warn("template discarded", logger.String("file", issue.File), logger.Error(issue.Err))
		}
		a.logger.Info("✅ Templates reloaded",
			logger.String("dir", a.cfg.TemplatesDir),
			logger.Int("loaded", a.templateCatalog.Count()),
			logger.Int("discarded", len(issues)))
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("template watcher stopped: %w", err)
	}
	return nil
}

func (a *App) consumeReporting(ctx context.Context) error {
	handler := func(ctx context.Context, event reporting.Event) error {
		if err := event.Validate(); err != nil {
			return err
		}
		if !a.registration.AuthenticateDispatcher(ctx, event.DispatcherID, event.RegistrationToken) {
			a.logger.Warn("ignored reporting from unregistered dispatcher",
				logger.String("dispatcher", event.DispatcherID))
			return nil
		}
		a.metrics.ObserveReporting(event)
		return nil
	}
	if err := unexpectedWorkerError("reporting consumer", a.reportingConsumer.Run(ctx, handler)); err != nil {
		return err
	}
	return nil
}

func (a *App) consumeHeartbeats(ctx context.Context) error {
	handler := func(ctx context.Context, event heartbeat.Event) error {
		accepted := a.registration.AuthenticateDispatcher(ctx, event.DispatcherID, event.RegistrationToken)
		a.publishHeartbeatAck(ctx, event, accepted)
		if !accepted {
			a.logger.Debug("ignored heartbeat from unregistered dispatcher",
				logger.String("dispatcher", event.DispatcherID))
			return nil
		}
		a.metrics.ObserveHeartbeat(event)
		return nil
	}
	if err := unexpectedWorkerError("heartbeat consumer", a.heartbeatConsumer.Run(ctx, handler)); err != nil {
		return err
	}
	return nil
}

func (a *App) cleanPlans(ctx context.Context) error {
	a.logger.Info("starting notification plan cleanup",
		logger.Duration("retention", a.cfg.PlanRetention),
		logger.Duration("interval", a.cfg.PlanCleanupInterval))
	return a.runStreamCleanup(ctx, "notification plans", a.planCleaner)
}

func (a *App) cleanReporting(ctx context.Context) error {
	a.logger.Info("starting reporting cleanup",
		logger.Duration("retention", a.cfg.ReportingRetention),
		logger.Duration("interval", a.cfg.ReportingCleanupInterval))
	return a.runStreamCleanup(ctx, "reporting events", a.reportingCleaner)
}

func (a *App) cleanLifecycle(ctx context.Context) error {
	a.logger.Info("starting lifecycle cleanup",
		logger.Duration("retention", redisstreams.LifecycleRetention),
		logger.Duration("interval", redisstreams.LifecycleCleanupInterval))
	return a.runStreamCleanup(ctx, "lifecycle events", a.lifecycleCleaner)
}

func unexpectedWorkerError(name string, err error) error {
	if err == nil || errors.Is(err, context.Canceled) {
		return nil
	}
	return fmt.Errorf("%s stopped: %w", name, err)
}

func (a *App) runStreamCleanup(ctx context.Context, name string, base *redisstreams.StreamCleaner) error {
	cleaner := *base
	cleaner.OnError = func(err error) {
		a.logger.Warn("stream cleanup failed", logger.String("stream", name), logger.Error(err))
	}
	cleaner.OnTrim = func(count int64) {
		a.logger.Info("stream entries trimmed", logger.String("stream", name), logger.Int64("count", count))
	}
	if err := cleaner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s cleanup stopped: %w", name, err)
	}
	return nil
}

func (a *App) publishHeartbeatAck(ctx context.Context, event heartbeat.Event, accepted bool) {
	ack := heartbeat.Ack{
		ID:           fmt.Sprintf("%s:heartbeat-ack:%d", a.coreID, time.Now().UnixNano()),
		HeartbeatID:  event.ID,
		DispatcherID: event.DispatcherID,
		CoreID:       a.coreID,
		OccurredAt:   time.Now().UTC(),
		Accepted:     accepted,
	}
	if !accepted {
		ack.Reason = "dispatcher registration is not valid"
	}
	if err := a.heartbeatAckPublisher.Publish(ctx, ack); err != nil {
		a.logger.Warn("publish heartbeat acknowledgement",
			logger.Error(err),
			logger.String("dispatcher", event.DispatcherID))
	}
}
