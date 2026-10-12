package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/navire-dev/navire/shared/logger"
	"github.com/navire-dev/navire/shared/models"
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

func (a *App) consumePlans(ctx context.Context) error {
	handler := func(ctx context.Context, plan models.ExecutionPlan) error {
		started := time.Now()
		a.logger.Info("dispatching execution plan",
			logger.String("message_id", plan.MessageID),
			logger.String("transport_message_id", plan.TransportMessageID),
			logger.String("target", plan.Target.Name),
			logger.String("provider", string(plan.Target.Provider)))
		if err := a.dispatcher.Dispatch(ctx, plan); err != nil {
			a.metrics.DeliveryAttempt(string(plan.Target.Provider), "failure")
			a.metrics.DeliveryDuration(string(plan.Target.Provider), time.Since(started))
			a.logger.Error("delivery failed",
				logger.String("message_id", plan.MessageID),
				logger.String("target", plan.Target.Name),
				logger.String("provider", string(plan.Target.Provider)),
				logger.Error(err))
			return nil
		}
		a.metrics.DeliveryAttempt(string(plan.Target.Provider), "success")
		a.metrics.DeliveryDuration(string(plan.Target.Provider), time.Since(started))
		a.logger.Info("execution plan delivered",
			logger.String("message_id", plan.MessageID),
			logger.String("target", plan.Target.Name),
			logger.String("provider", string(plan.Target.Provider)),
			logger.Duration("duration", time.Since(started)))
		return nil
	}
	if err := unexpectedWorkerError("plan consumer", a.planConsumer.Run(ctx, handler)); err != nil {
		return err
	}
	return nil
}

func (a *App) cleanPlans(ctx context.Context) error {
	a.logger.Info("starting notification plan cleanup",
		logger.Duration("retention", a.cfg.PlanRetention),
		logger.Duration("interval", a.cfg.PlanCleanupInterval))
	return a.runStreamCleanups(ctx, "notification plans", a.planCleaners)
}

func (a *App) runStreamCleanups(ctx context.Context, name string, cleaners []*redisstreams.StreamCleaner) error {
	if len(cleaners) == 0 {
		return fmt.Errorf("%s cleanup has no streams", name)
	}
	cleanupCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errs := make(chan error, len(cleaners))
	var workers sync.WaitGroup
	workers.Add(len(cleaners))
	for _, cleaner := range cleaners {
		go func(cleaner *redisstreams.StreamCleaner) {
			defer workers.Done()
			if err := a.runStreamCleanup(cleanupCtx, name, cleaner); err != nil {
				errs <- err
			}
		}(cleaner)
	}
	done := make(chan struct{})
	go func() {
		workers.Wait()
		close(done)
	}()
	select {
	case err := <-errs:
		cancel()
		<-done
		return err
	case <-done:
		return nil
	case <-ctx.Done():
		cancel()
		<-done
		return ctx.Err()
	}
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
