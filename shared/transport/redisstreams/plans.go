package redisstreams

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/navire-dev/navire/shared/models"
	"github.com/redis/go-redis/v9"
)

// PlanConsumer reads provider-partitioned execution plans and processes them
// with a bounded pool of sequential workers.
type PlanConsumer struct {
	Client    *redis.Client
	Streams   []string
	Group     string
	Consumer  string
	Workers   int
	Block     time.Duration
	OnConsume func(context.Context, models.ExecutionPlan)
	OnAck     func(context.Context, models.ExecutionPlan)
}

type planMessage struct {
	stream  string
	message redis.XMessage
}

// Run consumes plans until the context is canceled or a worker encounters a
// transport or handler error. The handler completes before the plan is ACKed.
func (c PlanConsumer) Run(ctx context.Context, handler Handler) error {
	if handler == nil {
		return fmt.Errorf("plan consumer handler is required")
	}
	if err := c.validate(); err != nil {
		return err
	}
	group := c.Group
	if group == "" {
		group = NotificationGroup
	}
	block := c.Block
	if block <= 0 {
		block = 5 * time.Second
	}
	workers := c.Workers
	if workers <= 0 {
		workers = 1
	}
	streams := uniqueStreams(c.Streams)
	for _, stream := range streams {
		if err := ensureConsumerGroup(ctx, c.Client, stream, group, "0"); err != nil {
			return err
		}
	}

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan planMessage, workers)
	workerErrors := make(chan error, 1)
	var workersDone sync.WaitGroup
	workersDone.Add(workers)
	for range workers {
		go func() {
			defer workersDone.Done()
			for {
				select {
				case <-workerCtx.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					if err := handlePlanMessage(workerCtx, c.Client, group, job, handler, c.OnConsume, c.OnAck); err != nil {
						select {
						case workerErrors <- err:
						default:
						}
						cancel()
						return
					}
				}
			}
		}()
	}

	readerErr := readPlans(workerCtx, c.Client, streams, group, c.Consumer, block, jobs)
	cancel()
	close(jobs)
	workersDone.Wait()

	select {
	case err := <-workerErrors:
		return err
	default:
	}
	if readerErr != nil && !errors.Is(readerErr, context.Canceled) {
		return readerErr
	}
	return ctx.Err()
}

func (c PlanConsumer) validate() error {
	if c.Client == nil {
		return fmt.Errorf("redis client is required")
	}
	if len(uniqueStreams(c.Streams)) == 0 {
		return fmt.Errorf("at least one plan stream is required")
	}
	if c.Consumer == "" {
		return fmt.Errorf("consumer name is required")
	}
	if c.Workers < 0 {
		return fmt.Errorf("worker count cannot be negative")
	}
	return nil
}

func readPlans(ctx context.Context, client *redis.Client, streams []string, group, consumer string, block time.Duration, jobs chan<- planMessage) error {
	start := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		ordered := rotateStreams(streams, start)
		keys := make([]string, 0, len(ordered)*2)
		keys = append(keys, ordered...)
		for range ordered {
			keys = append(keys, ">")
		}
		entries, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams:  keys,
			Count:    1,
			Block:    block,
			NoAck:    false,
		}).Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read execution plans: %w", err)
		}
		for _, entry := range entries {
			for _, message := range entry.Messages {
				start = (streamIndex(streams, entry.Stream) + 1) % len(streams)
				select {
				case jobs <- planMessage{stream: entry.Stream, message: message}:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}

func handlePlanMessage(ctx context.Context, client *redis.Client, group string, job planMessage, handler Handler, onConsume, onAck func(context.Context, models.ExecutionPlan)) error {
	raw, ok := job.message.Values["payload"].(string)
	if !ok {
		if payload, valid := job.message.Values["payload"].([]byte); valid {
			raw = string(payload)
		}
	}
	var plan models.ExecutionPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return fmt.Errorf("decode execution plan %s: %w", job.message.ID, err)
	}
	plan.TransportMessageID = job.message.ID
	if onConsume != nil {
		onConsume(ctx, plan)
	}
	if err := handler(ctx, plan); err != nil {
		return fmt.Errorf("process execution plan %s: %w", job.message.ID, err)
	}
	if err := client.XAck(ctx, job.stream, group, job.message.ID).Err(); err != nil {
		return fmt.Errorf("ack execution plan %s: %w", job.message.ID, err)
	}
	if onAck != nil {
		onAck(ctx, plan)
	}
	return nil
}

func uniqueStreams(streams []string) []string {
	seen := make(map[string]struct{}, len(streams))
	result := make([]string, 0, len(streams))
	for _, stream := range streams {
		if stream == "" {
			continue
		}
		if _, exists := seen[stream]; exists {
			continue
		}
		seen[stream] = struct{}{}
		result = append(result, stream)
	}
	return result
}

func rotateStreams(streams []string, start int) []string {
	ordered := make([]string, 0, len(streams))
	ordered = append(ordered, streams[start:]...)
	ordered = append(ordered, streams[:start]...)
	return ordered
}

func streamIndex(streams []string, target string) int {
	for index, stream := range streams {
		if stream == target {
			return index
		}
	}
	return 0
}
