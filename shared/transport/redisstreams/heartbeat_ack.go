package redisstreams

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/navire-dev/navire/shared/heartbeat"
	"github.com/redis/go-redis/v9"
)

const heartbeatAckStreamPrefix = "navire:heartbeat-ack:"

func HeartbeatAckStream(dispatcherID string) string {
	return heartbeatAckStreamPrefix + strings.TrimSpace(dispatcherID)
}

func HeartbeatAckGroup(dispatcherID string) string {
	return "navire-dspc-heartbeat-ack:" + strings.TrimSpace(dispatcherID)
}

type HeartbeatAckPublisher struct {
	Client *redis.Client
}

func (p HeartbeatAckPublisher) Publish(ctx context.Context, ack heartbeat.Ack) error {
	if p.Client == nil {
		return fmt.Errorf("redis client is required")
	}
	if err := ack.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(ack)
	if err != nil {
		return fmt.Errorf("encode heartbeat acknowledgement: %w", err)
	}
	if err := p.Client.XAdd(ctx, &redis.XAddArgs{
		Stream: HeartbeatAckStream(ack.DispatcherID),
		Values: map[string]any{"payload": payload},
	}).Err(); err != nil {
		return fmt.Errorf("publish heartbeat acknowledgement: %w", err)
	}
	return nil
}

type HeartbeatAckHandler func(context.Context, heartbeat.Ack) error

type HeartbeatAckConsumer struct {
	Client       *redis.Client
	DispatcherID string
	Block        time.Duration
}

func (c HeartbeatAckConsumer) Run(ctx context.Context, handler HeartbeatAckHandler) error {
	if handler == nil {
		return fmt.Errorf("heartbeat acknowledgement handler is required")
	}
	if c.Block <= 0 {
		c.Block = 5 * time.Second
	}
	stream := HeartbeatAckStream(c.DispatcherID)
	group := HeartbeatAckGroup(c.DispatcherID)
	if err := validateConsumerGroupConfig(c.Client, stream, group, c.DispatcherID); err != nil {
		return err
	}
	if err := ensureConsumerGroup(ctx, c.Client, stream, group, "$"); err != nil {
		return err
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		messages, err := readConsumerMessages(ctx, c.Client, stream, group, c.DispatcherID, 10, c.Block)
		if err != nil {
			return fmt.Errorf("read heartbeat acknowledgements: %w", err)
		}
		for _, message := range messages {
			if err := handleHeartbeatAck(ctx, c.Client, stream, group, message, handler); err != nil {
				return err
			}
		}
	}
}

func handleHeartbeatAck(ctx context.Context, client *redis.Client, stream, group string, message redis.XMessage, handler HeartbeatAckHandler) error {
	raw, ok := message.Values["payload"].(string)
	if !ok {
		if payload, valid := message.Values["payload"].([]byte); valid {
			raw = string(payload)
		}
	}
	var ack heartbeat.Ack
	if err := json.Unmarshal([]byte(raw), &ack); err != nil {
		return fmt.Errorf("decode heartbeat acknowledgement %s: %w", message.ID, err)
	}
	if err := ack.Validate(); err != nil {
		return fmt.Errorf("validate heartbeat acknowledgement %s: %w", message.ID, err)
	}
	if err := handler(ctx, ack); err != nil {
		return fmt.Errorf("process heartbeat acknowledgement %s: %w", message.ID, err)
	}
	return acknowledgeAndDelete(ctx, client, stream, group, message.ID)
}
