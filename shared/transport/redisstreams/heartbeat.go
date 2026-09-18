package redisstreams

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/navire-dev/navire/shared/heartbeat"
	"github.com/redis/go-redis/v9"
)

const (
	HeartbeatStream = "navire:heartbeat"
	HeartbeatGroup  = "navire-core-heartbeat"
)

type HeartbeatPublisher struct {
	Client *redis.Client
	Stream string
}

func (p HeartbeatPublisher) Publish(ctx context.Context, event heartbeat.Event) error {
	if p.Client == nil {
		return fmt.Errorf("redis client is required")
	}
	if err := event.Validate(); err != nil {
		return err
	}
	stream := p.Stream
	if stream == "" {
		stream = HeartbeatStream
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode heartbeat event: %w", err)
	}
	if err := p.Client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{"payload": payload},
	}).Err(); err != nil {
		return fmt.Errorf("publish heartbeat event: %w", err)
	}
	return nil
}

type HeartbeatHandler func(context.Context, heartbeat.Event) error

type HeartbeatConsumer struct {
	Client   *redis.Client
	Stream   string
	Group    string
	Consumer string
	Block    time.Duration
}

func (c HeartbeatConsumer) Run(ctx context.Context, handler HeartbeatHandler) error {
	if handler == nil {
		return fmt.Errorf("heartbeat consumer handler is required")
	}
	if c.Stream == "" {
		c.Stream = HeartbeatStream
	}
	if c.Group == "" {
		c.Group = HeartbeatGroup
	}
	if c.Block <= 0 {
		c.Block = 5 * time.Second
	}
	if err := validateConsumerGroupConfig(c.Client, c.Stream, c.Group, c.Consumer); err != nil {
		return err
	}
	if err := ensureConsumerGroup(ctx, c.Client, c.Stream, c.Group, "0"); err != nil {
		return err
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		messages, err := readConsumerMessages(ctx, c.Client, c.Stream, c.Group, c.Consumer, 100, c.Block)
		if err != nil {
			return fmt.Errorf("read heartbeat events: %w", err)
		}
		for _, message := range messages {
			if err := handleHeartbeat(ctx, c.Client, c.Stream, c.Group, message, handler); err != nil {
				return err
			}
		}
	}
}

func handleHeartbeat(ctx context.Context, client *redis.Client, stream, group string, message redis.XMessage, handler HeartbeatHandler) error {
	raw, ok := message.Values["payload"].(string)
	if !ok {
		if payload, valid := message.Values["payload"].([]byte); valid {
			raw = string(payload)
		}
	}
	var event heartbeat.Event
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		return fmt.Errorf("decode heartbeat event %s: %w", message.ID, err)
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate heartbeat event %s: %w", message.ID, err)
	}
	if err := handler(ctx, event); err != nil {
		return fmt.Errorf("process heartbeat event %s: %w", message.ID, err)
	}
	return acknowledgeAndDelete(ctx, client, stream, group, message.ID)
}
