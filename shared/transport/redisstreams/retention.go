package redisstreams

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TrimBefore removes stream entries older than cutoff. Redis stream IDs use
// Unix milliseconds as their first component, which makes them suitable for
// time-based retention.
func TrimBefore(ctx context.Context, client *redis.Client, stream string, cutoff time.Time) (int64, error) {
	if client == nil {
		return 0, fmt.Errorf("redis client is required")
	}
	if stream == "" {
		return 0, fmt.Errorf("stream is required")
	}
	minID := fmt.Sprintf("%d-0", cutoff.UnixMilli())
	trimmed, err := client.XTrimMinID(ctx, stream, minID).Result()
	if err != nil {
		return 0, fmt.Errorf("trim stream %s before %s: %w", stream, minID, err)
	}
	return trimmed, nil
}

// StreamCleaner periodically removes expired entries from a Redis stream.
// Stream retention is a hard maximum lifetime for the stream's entries.
type StreamCleaner struct {
	Client    *redis.Client
	Stream    string
	Retention time.Duration
	Interval  time.Duration
	OnError   func(error)
	OnTrim    func(int64)
}

func (c StreamCleaner) Run(ctx context.Context) error {
	if c.Client == nil {
		return fmt.Errorf("redis client is required")
	}
	if c.Stream == "" {
		return fmt.Errorf("stream is required")
	}
	if c.Retention <= 0 {
		return fmt.Errorf("stream retention must be greater than zero")
	}
	if c.Interval <= 0 {
		return fmt.Errorf("plan cleanup interval must be greater than zero")
	}

	c.trimAndReport(ctx)

	ticker := time.NewTicker(c.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			c.trimAndReport(ctx)
		}
	}
}

func (c StreamCleaner) trimAndReport(ctx context.Context) {
	trimmed, err := c.trim(ctx)
	if err != nil {
		if ctx.Err() == nil && c.OnError != nil {
			c.OnError(err)
		}
		return
	}
	if trimmed > 0 && c.OnTrim != nil {
		c.OnTrim(trimmed)
	}
}

func (c StreamCleaner) trim(ctx context.Context) (int64, error) {
	cutoff := time.Now().UTC().Add(-c.Retention)
	return TrimBefore(ctx, c.Client, c.Stream, cutoff)
}

func acknowledgeAndDelete(ctx context.Context, client *redis.Client, stream, group, messageID string) error {
	var ack *redis.IntCmd
	var deleted *redis.IntCmd
	if _, err := client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		ack = pipe.XAck(ctx, stream, group, messageID)
		deleted = pipe.XDel(ctx, stream, messageID)
		return nil
	}); err != nil {
		return fmt.Errorf("ack and delete stream entry %s: %w", messageID, err)
	}
	if err := ack.Err(); err != nil {
		return fmt.Errorf("ack stream entry %s: %w", messageID, err)
	}
	if err := deleted.Err(); err != nil {
		return fmt.Errorf("delete stream entry %s: %w", messageID, err)
	}
	return nil
}
