package metrics

import (
	"context"
	"time"
)

// Recorder is the Core-owned metrics contract. Provider-specific exporters
// implement it without leaking their client library into Core business code.
type Recorder interface {
	SetRedisConnected(bool)
	RedisOperationError(operation string)
	EventReceived()
	EventAccepted()
	EventRejected(reason string)
	Event(template, variant, state string)
	EventPublished()
	EventPublishError()
	StreamPublished(count int)
	StreamConsumed(provider string)
	StreamAcked(provider string)
	DeliveryAttempt(provider, state string)
	DeliveryDuration(provider string, duration time.Duration)
}

// Exporter renders the Core-owned metrics for one external monitoring system.
// Prometheus is the first implementation; future exporters can implement the
// same boundary without changing API handlers or business services.
type Exporter interface {
	Scrape(context.Context) (Payload, error)
}

type Payload struct {
	Body        []byte
	ContentType string
}
