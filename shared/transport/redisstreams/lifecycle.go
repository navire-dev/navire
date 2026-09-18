package redisstreams

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/navire-dev/navire/shared/lifecycle"
	"github.com/redis/go-redis/v9"
)

const (
	LifecycleStream          = "navire:lifecycle"
	LifecycleRetention       = time.Hour
	LifecycleCleanupInterval = time.Hour
)

const lifecycleGroupPrefix = "navire-dspc-lifecycle"

func LifecycleGroup(dispatcherID string) string {
	return lifecycleGroupPrefix + ":" + strings.TrimSpace(dispatcherID)
}

type LifecyclePublisher struct {
	Client *redis.Client
	Stream string
}

func (p LifecyclePublisher) Publish(ctx context.Context, event lifecycle.Event) (string, error) {
	if p.Client == nil {
		return "", fmt.Errorf("redis client is required")
	}
	if err := event.Validate(); err != nil {
		return "", err
	}
	stream := p.Stream
	if stream == "" {
		stream = LifecycleStream
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("encode lifecycle event: %w", err)
	}
	id, err := p.Client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{"payload": payload},
	}).Result()
	if err != nil {
		return "", fmt.Errorf("publish lifecycle event: %w", err)
	}
	return id, nil
}

type LifecycleHandler func(context.Context, lifecycle.Event) error

type LifecycleConsumer struct {
	Client   *redis.Client
	Stream   string
	Group    string
	Consumer string
	Block    time.Duration
	OnAck    func(context.Context, lifecycle.Event)
}

func (c LifecycleConsumer) Run(ctx context.Context, handler LifecycleHandler) error {
	if handler == nil {
		return fmt.Errorf("lifecycle handler is required")
	}
	if c.Stream == "" {
		c.Stream = LifecycleStream
	}
	if c.Block <= 0 {
		c.Block = 5 * time.Second
	}
	if err := validateConsumerGroupConfig(c.Client, c.Stream, c.Group, c.Consumer); err != nil {
		return err
	}

	if err := ensureConsumerGroup(ctx, c.Client, c.Stream, c.Group, "$"); err != nil {
		return err
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		messages, err := readConsumerMessages(ctx, c.Client, c.Stream, c.Group, c.Consumer, 10, c.Block)
		if err != nil {
			return fmt.Errorf("read lifecycle events: %w", err)
		}
		for _, message := range messages {
			if err := handleLifecycleMessage(ctx, c.Client, c.Stream, c.Group, message, handler, c.OnAck); err != nil {
				return err
			}
		}
	}
}

func handleLifecycleMessage(ctx context.Context, client *redis.Client, stream, group string, message redis.XMessage, handler LifecycleHandler, onAck func(context.Context, lifecycle.Event)) error {
	raw, ok := message.Values["payload"].(string)
	if !ok {
		if payload, valid := message.Values["payload"].([]byte); valid {
			raw = string(payload)
		}
	}
	var event lifecycle.Event
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		return fmt.Errorf("decode lifecycle event %s: %w", message.ID, err)
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate lifecycle event %s: %w", message.ID, err)
	}
	if err := handler(ctx, event); err != nil {
		return fmt.Errorf("process lifecycle event %s: %w", message.ID, err)
	}
	if err := client.XAck(ctx, stream, group, message.ID).Err(); err != nil {
		return fmt.Errorf("ack lifecycle event %s: %w", message.ID, err)
	}
	if onAck != nil {
		onAck(ctx, event)
	}
	return nil
}
