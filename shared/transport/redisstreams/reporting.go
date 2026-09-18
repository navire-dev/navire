package redisstreams

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/navire-dev/navire/shared/reporting"
	"github.com/redis/go-redis/v9"
)

const (
	ReportingStream = "navire:reporting"
	ReportingGroup  = "navire-core-reporting"
)

type ReportingPublisher struct {
	Client *redis.Client
	Stream string
}

func (p ReportingPublisher) Publish(ctx context.Context, event reporting.Event) error {
	if p.Client == nil {
		return fmt.Errorf("redis client is required")
	}
	if err := event.Validate(); err != nil {
		return err
	}
	if p.Stream == "" {
		p.Stream = ReportingStream
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode reporting event: %w", err)
	}
	if err := p.Client.XAdd(ctx, &redis.XAddArgs{
		Stream: p.Stream,
		Values: map[string]any{"payload": payload},
	}).Err(); err != nil {
		return fmt.Errorf("publish reporting event: %w", err)
	}
	return nil
}

type ReportingHandler func(context.Context, reporting.Event) error

type ReportingConsumer struct {
	Client   *redis.Client
	Stream   string
	Group    string
	Consumer string
	Block    time.Duration
}

func (c ReportingConsumer) Run(ctx context.Context, handler ReportingHandler) error {
	if handler == nil {
		return fmt.Errorf("reporting consumer handler is required")
	}
	if c.Stream == "" {
		c.Stream = ReportingStream
	}
	if c.Group == "" {
		c.Group = ReportingGroup
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
			return fmt.Errorf("read reporting events: %w", err)
		}
		for _, message := range messages {
			if err := handleReportingMessage(ctx, c.Client, c.Stream, c.Group, message, handler); err != nil {
				return err
			}
		}
	}
}

func handleReportingMessage(ctx context.Context, client *redis.Client, stream, group string, message redis.XMessage, handler ReportingHandler) error {
	raw, ok := message.Values["payload"].(string)
	if !ok {
		if bytes, valid := message.Values["payload"].([]byte); valid {
			raw = string(bytes)
		}
	}
	var event reporting.Event
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		return fmt.Errorf("decode reporting event %s: %w", message.ID, err)
	}
	if err := handler(ctx, event); err != nil {
		return fmt.Errorf("process reporting event %s: %w", message.ID, err)
	}
	return acknowledgeAndDelete(ctx, client, stream, group, message.ID)
}
