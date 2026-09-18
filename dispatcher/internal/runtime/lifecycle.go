package runtime

import (
	"context"
	"sync"

	"github.com/navire-dev/navire/shared/logger"
	sharedruntime "github.com/navire-dev/navire/shared/runtime"
)

func (a *App) Run() error {
	ctx, stop := sharedruntime.SignalContext()
	defer stop()
	return a.RunContext(ctx)
}

func (a *App) RunContext(ctx context.Context) error {
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()
	var workers sync.WaitGroup
	startWorker := func(worker func(context.Context)) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			worker(workerCtx)
		}()
	}
	startWorker(a.registerUntilAvailable)
	startWorker(a.consumeUntilAvailable)
	startWorker(a.publishHeartbeats)
	<-ctx.Done()
	a.logger.Info("⏳ Dispatcher shutting down gracefully...")
	cancelWorkers()
	workers.Wait()
	if err := a.redis.Close(); err != nil {
		a.logger.Warn("dispatcher Redis client close failed", logger.Error(err))
	}
	a.logger.Info("✅ Navire dispatcher stopped cleanly")
	return nil
}
