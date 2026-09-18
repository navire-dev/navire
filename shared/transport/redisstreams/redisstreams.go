package redisstreams

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/navire-dev/navire/shared/models"
	sharedtransport "github.com/navire-dev/navire/shared/transport"
	"github.com/redis/go-redis/v9"
)

const (
	NotificationStream = "navire:notifications"
	NotificationGroup  = "navire-dispatchers"
)

type Publisher struct {
	Client *redis.Client
	Stream string
}

func (p Publisher) Publish(ctx context.Context, plan models.ExecutionPlan) (sharedtransport.MessageID, error) {
	ids, err := p.PublishMany(ctx, []models.ExecutionPlan{plan})
	if err != nil {
		return "", err
	}
	if len(ids) != 1 {
		return "", fmt.Errorf("publish returned %d IDs for one execution plan", len(ids))
	}
	return ids[0], nil
}

// PublishMany atomically publishes one execution plan per delivery target.
func (p Publisher) PublishMany(ctx context.Context, plans []models.ExecutionPlan) ([]sharedtransport.MessageID, error) {
	if p.Client == nil {
		return nil, fmt.Errorf("redis client is required")
	}
	if len(plans) == 0 {
		return nil, fmt.Errorf("at least one execution plan is required")
	}
	if p.Stream == "" {
		p.Stream = NotificationStream
	}
	payloads := make([][]byte, len(plans))
	for i, plan := range plans {
		payload, err := json.Marshal(plan)
		if err != nil {
			return nil, fmt.Errorf("encode execution plan %d: %w", i, err)
		}
		payloads[i] = payload
	}
	commands := make([]*redis.StringCmd, len(plans))
	_, err := p.Client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		for i, payload := range payloads {
			commands[i] = pipe.XAdd(ctx, &redis.XAddArgs{
				Stream: p.Stream,
				Values: map[string]any{"payload": payload},
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("publish execution plans: %w", err)
	}
	ids := make([]sharedtransport.MessageID, len(commands))
	for i, command := range commands {
		id, err := command.Result()
		if err != nil {
			return nil, fmt.Errorf("read published execution plan %d ID: %w", i, err)
		}
		ids[i] = sharedtransport.MessageID(id)
	}
	return ids, nil
}

type Handler func(context.Context, models.ExecutionPlan) error

type Consumer struct {
	Client   *redis.Client
	Stream   string
	Group    string
	Consumer string
	Block    time.Duration
	OnAck    func(context.Context, models.ExecutionPlan)
}

func (c Consumer) Lag(ctx context.Context) (int64, error) {
	if c.Client == nil {
		return -1, fmt.Errorf("redis client is required")
	}
	stream := c.Stream
	if stream == "" {
		stream = NotificationStream
	}
	group := c.Group
	if group == "" {
		group = NotificationGroup
	}
	groups, err := c.Client.XInfoGroups(ctx, stream).Result()
	if err != nil {
		return -1, fmt.Errorf("read consumer lag: %w", err)
	}
	for _, info := range groups {
		if info.Name == group {
			return info.Lag, nil
		}
	}
	return -1, nil
}

func (c Consumer) Run(ctx context.Context, handler Handler) error {
	if handler == nil {
		return fmt.Errorf("consumer handler is required")
	}
	if c.Stream == "" {
		c.Stream = NotificationStream
	}
	if c.Group == "" {
		c.Group = NotificationGroup
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
		messages, err := readConsumerMessages(ctx, c.Client, c.Stream, c.Group, c.Consumer, 10, c.Block)
		if err != nil {
			return fmt.Errorf("read execution plans: %w", err)
		}
		for _, message := range messages {
			if err := handleMessage(ctx, c.Client, c.Stream, c.Group, message, handler, c.OnAck); err != nil {
				return err
			}
		}
	}
}

func handleMessage(ctx context.Context, client *redis.Client, stream, group string, message redis.XMessage, handler Handler, onAck func(context.Context, models.ExecutionPlan)) error {
	raw, ok := message.Values["payload"].(string)
	if !ok {
		if bytes, valid := message.Values["payload"].([]byte); valid {
			raw = string(bytes)
		}
	}
	var plan models.ExecutionPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return fmt.Errorf("decode execution plan %s: %w", message.ID, err)
	}
	plan.TransportMessageID = message.ID
	if err := handler(ctx, plan); err != nil {
		return fmt.Errorf("process execution plan %s: %w", message.ID, err)
	}
	if err := client.XAck(ctx, stream, group, message.ID).Err(); err != nil {
		return fmt.Errorf("ack execution plan %s: %w", message.ID, err)
	}
	if onAck != nil {
		onAck(ctx, plan)
	}
	return nil
}
