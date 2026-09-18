package runtime

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// SignalContext returns a context cancelled by the standard process signals.
func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

// NextRetryDelay doubles delay while respecting its upper bound.
func NextRetryDelay(delay, limit time.Duration) time.Duration {
	if delay >= limit {
		return limit
	}
	next := delay * 2
	if next > limit {
		return limit
	}
	return next
}

// WaitForRetry waits for delay or returns early when ctx is cancelled.
func WaitForRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
