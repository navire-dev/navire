package redisstreams

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func ensureConsumerGroup(ctx context.Context, client *redis.Client, stream, group, startID string) error {
	err := client.XGroupCreateMkStream(ctx, stream, group, startID).Err()
	if err == nil || redis.HasErrorPrefix(err, "BUSYGROUP") {
		return nil
	}

	return fmt.Errorf("create consumer group %q on stream %q: %w", group, stream, err)
}

func readConsumerMessages(ctx context.Context, client *redis.Client, stream, group, consumer string, count int64, block time.Duration) ([]redis.XMessage, error) {
	entries, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    count,
		Block:    block,
		NoAck:    false,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return flattenConsumerMessages(entries), nil
}

func flattenConsumerMessages(entries []redis.XStream) []redis.XMessage {
	messages := make([]redis.XMessage, 0)
	for _, entry := range entries {
		messages = append(messages, entry.Messages...)
	}
	return messages
}

func validateConsumerGroupConfig(client *redis.Client, stream, group, consumer string) error {
	if client == nil {
		return fmt.Errorf("redis client is required")
	}
	if stream == "" {
		return fmt.Errorf("stream is required")
	}
	if group == "" {
		return fmt.Errorf("consumer group is required")
	}
	if consumer == "" {
		return fmt.Errorf("consumer name is required")
	}
	return nil
}
