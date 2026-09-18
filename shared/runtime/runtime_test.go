package runtime

import (
	"context"
	"testing"
	"time"
)

func TestNextRetryDelay(t *testing.T) {
	tests := []struct {
		name    string
		current time.Duration
		limit   time.Duration
		want    time.Duration
	}{
		{name: "doubles", current: 10 * time.Second, limit: 30 * time.Second, want: 20 * time.Second},
		{name: "caps", current: 20 * time.Second, limit: 30 * time.Second, want: 30 * time.Second},
		{name: "stays capped", current: 30 * time.Second, limit: 30 * time.Second, want: 30 * time.Second},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NextRetryDelay(test.current, test.limit); got != test.want {
				t.Fatalf("NextRetryDelay() = %s, want %s", got, test.want)
			}
		})
	}
}

func TestWaitForRetryStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if WaitForRetry(ctx, time.Hour) {
		t.Fatal("WaitForRetry() = true, want false")
	}
}
