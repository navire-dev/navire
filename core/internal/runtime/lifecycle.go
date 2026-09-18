package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/navire-dev/navire/shared/lifecycle"
	"github.com/navire-dev/navire/shared/logger"
	sharedruntime "github.com/navire-dev/navire/shared/runtime"
)

func (a *App) Run() error {
	ctx, stop := sharedruntime.SignalContext()
	defer stop()
	return a.RunContext(ctx)
}

func startWorker(
	workers *sync.WaitGroup,
	workerErrCh chan error,
	workerCtx context.Context,
	name string,
	worker func(context.Context) error,
) {
	workers.Add(1)
	go func() {
		defer workers.Done()

		err := worker(workerCtx)

		if err == nil && workerCtx.Err() == nil {
			err = fmt.Errorf("%s stopped unexpectedly", name)
		}

		if err == nil || errors.Is(err, context.Canceled) {
			return
		}

		select {
		case workerErrCh <- fmt.Errorf("%s: %w", name, err):
		default:
		}
	}()
}

func (a *App) RunContext(ctx context.Context) error {
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()
	var workers sync.WaitGroup
	workerErrCh := make(chan error, 1)
	startWorker(&workers, workerErrCh, workerCtx, "config worker", a.watchConfig)
	startWorker(&workers, workerErrCh, workerCtx, "template worker", a.watchTemplates)
	startWorker(&workers, workerErrCh, workerCtx, "reporting worker", a.consumeReporting)
	startWorker(&workers, workerErrCh, workerCtx, "heartbeat worker", a.consumeHeartbeats)
	startWorker(&workers, workerErrCh, workerCtx, "plan cleanup worker", a.cleanPlans)
	startWorker(&workers, workerErrCh, workerCtx, "reporting cleanup worker", a.cleanReporting)
	startWorker(&workers, workerErrCh, workerCtx, "lifecycle cleanup worker", a.cleanLifecycle)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- a.server.Start()
	}()

	var serverErr error
	select {
	case <-ctx.Done():
		a.logger.Info("⏳ Shutting down gracefully...")
	case serverErr = <-serverErrCh:
	case serverErr = <-workerErrCh:
		a.logger.Error("Core worker stopped unexpectedly", logger.Error(serverErr))
	}
	cancelWorkers()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout())
	defer cancel()
	a.publishShutdown(shutdownCtx, serverErr)

	if err := a.server.Stop(shutdownCtx); err != nil {
		workers.Wait()
		if serverErr != nil && !isContextError(serverErr) {
			return fmt.Errorf("server error: %w; shutdown error: %w", serverErr, err)
		}
		return fmt.Errorf("failed to stop server: %w", err)
	}
	workers.Wait()

	a.logger.Info("✅ Navire core stopped cleanly")
	if serverErr != nil && !isContextError(serverErr) {
		return serverErr
	}
	return nil
}

func (a *App) shutdownTimeout() time.Duration {
	if a.cfg.ShutdownTimeout > 0 {
		return a.cfg.ShutdownTimeout
	}
	return 10 * time.Second
}

func (a *App) publishShutdown(ctx context.Context, serverErr error) {
	event := lifecycle.Event{
		ID:         fmt.Sprintf("%s:shutdown:%d", a.coreID, time.Now().UnixNano()),
		Kind:       lifecycle.CoreShutdown,
		OccurredAt: time.Now().UTC(),
		CoreID:     a.coreID,
		Reason:     shutdownReason(serverErr),
	}
	if _, err := a.lifecyclePublisher.Publish(ctx, event); err != nil {
		a.logger.Warn("publish Core shutdown event", logger.Error(err))
	}
}

func shutdownReason(err error) string {
	if err == nil || isContextError(err) {
		return "context_canceled"
	}
	return "server_error"
}

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
