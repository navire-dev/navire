package metrics

import (
	"context"

	"github.com/navire-dev/navire/shared/heartbeat"
	"github.com/navire-dev/navire/shared/reporting"
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
	ObserveReporting(reporting.Event)
	ObserveHeartbeat(heartbeat.Event)
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
