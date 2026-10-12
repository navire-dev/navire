package prometheus

import (
	"bytes"
	"context"
	"fmt"
	"time"

	coremetrics "github.com/navire-dev/navire/core/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/common/expfmt"
)

// Metrics is the Prometheus implementation of the Core metrics contract.
// It records facts observed directly by the Core runtime.
type Metrics struct {
	registry *prometheus.Registry
	started  time.Time

	up                 prometheus.Gauge
	redisConnected     prometheus.Gauge
	redisOperationErrs *prometheus.CounterVec
	eventsReceived     prometheus.Counter
	eventsAccepted     prometheus.Counter
	eventsRejected     *prometheus.CounterVec
	events             *prometheus.CounterVec
	eventsPublished    prometheus.Counter
	eventsPublishErrs  prometheus.Counter
	streamPublished    prometheus.Counter
	streamConsumed     *prometheus.CounterVec
	streamAcked        *prometheus.CounterVec
	deliveryAttempts   *prometheus.CounterVec
	deliveryDuration   *prometheus.HistogramVec
}

func New(started time.Time) *Metrics {
	labels := prometheus.Labels{"component": "core"}
	registry := prometheus.NewRegistry()
	m := &Metrics{
		registry: registry,
		started:  started,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "navire_up", Help: "Whether Core is serving metrics.", ConstLabels: labels,
		}),
		redisConnected: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "navire_redis_connected", Help: "Whether Core can reach Redis.", ConstLabels: labels,
		}),
		redisOperationErrs: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "navire_redis_operation_errors_total", Help: "Redis operation errors.", ConstLabels: labels,
		}, []string{"operation"}),
		eventsReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "navire_events_received_total", Help: "Notifications received by Core.", ConstLabels: labels,
		}),
		eventsAccepted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "navire_events_accepted_total", Help: "Notifications durably accepted by Core.", ConstLabels: labels,
		}),
		eventsRejected: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "navire_events_rejected_total", Help: "Notifications rejected by Core.", ConstLabels: labels,
		}, []string{"reason"}),
		events: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "navire_events_total", Help: "Accepted events by template, variant and state.", ConstLabels: labels,
		}, []string{"template", "variant", "state"}),
		eventsPublished: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "navire_events_published_total", Help: "Events published by Core.", ConstLabels: labels,
		}),
		eventsPublishErrs: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "navire_events_publish_errors_total", Help: "Event publication errors.", ConstLabels: labels,
		}),
		streamPublished: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "navire_stream_messages_published_total", Help: "Notification plans published by Core.", ConstLabels: labels,
		}),
		streamConsumed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "navire_stream_messages_consumed_total", Help: "Notification plans consumed by Core.", ConstLabels: labels,
		}, []string{"provider"}),
		streamAcked: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "navire_stream_messages_acked_total", Help: "Notification plans acknowledged by Core.", ConstLabels: labels,
		}, []string{"provider"}),
		deliveryAttempts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "navire_delivery_attempts_total", Help: "Delivery attempts by provider and state.", ConstLabels: labels,
		}, []string{"provider", "state"}),
		deliveryDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "navire_delivery_duration_seconds", Help: "Delivery duration by provider.", ConstLabels: labels,
		}, []string{"provider"}),
	}
	m.up.Set(1)
	registry.MustRegister(
		m.up,
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "navire_uptime_seconds", Help: "Core uptime in seconds.", ConstLabels: labels,
		}, func() float64 { return time.Since(m.started).Seconds() }),
		m.redisConnected,
		m.redisOperationErrs,
		m.eventsReceived,
		m.eventsAccepted,
		m.eventsRejected,
		m.events,
		m.eventsPublished,
		m.eventsPublishErrs,
		m.streamPublished,
		m.streamConsumed,
		m.streamAcked,
		m.deliveryAttempts,
		m.deliveryDuration,
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)
	return m
}

func (m *Metrics) Scrape(_ context.Context) (coremetrics.Payload, error) {
	metricFamilies, err := m.registry.Gather()
	if err != nil {
		return coremetrics.Payload{}, fmt.Errorf("gather Prometheus metrics: %w", err)
	}
	var body bytes.Buffer
	format := expfmt.NewFormat(expfmt.TypeTextPlain)
	encoder := expfmt.NewEncoder(&body, format)
	for _, family := range metricFamilies {
		if err := encoder.Encode(family); err != nil {
			return coremetrics.Payload{}, fmt.Errorf("encode Prometheus metrics: %w", err)
		}
	}
	return coremetrics.Payload{Body: body.Bytes(), ContentType: string(format)}, nil
}

func (m *Metrics) SetRedisConnected(connected bool) {
	if connected {
		m.redisConnected.Set(1)
		return
	}
	m.redisConnected.Set(0)
}

func (m *Metrics) RedisOperationError(operation string) {
	m.redisOperationErrs.WithLabelValues(operation).Inc()
}

func (m *Metrics) EventReceived() { m.eventsReceived.Inc() }

func (m *Metrics) EventAccepted() { m.eventsAccepted.Inc() }

func (m *Metrics) EventRejected(reason string) { m.eventsRejected.WithLabelValues(reason).Inc() }

func (m *Metrics) Event(template, variant, state string) {
	m.events.WithLabelValues(template, variant, state).Inc()
}

func (m *Metrics) EventPublished() { m.eventsPublished.Inc() }

func (m *Metrics) EventPublishError() { m.eventsPublishErrs.Inc() }

func (m *Metrics) StreamPublished(count int) { m.streamPublished.Add(float64(count)) }

func (m *Metrics) StreamConsumed(provider string) {
	m.streamConsumed.WithLabelValues(provider).Inc()
}

func (m *Metrics) StreamAcked(provider string) {
	m.streamAcked.WithLabelValues(provider).Inc()
}

func (m *Metrics) DeliveryAttempt(provider, state string) {
	m.deliveryAttempts.WithLabelValues(provider, state).Inc()
}

func (m *Metrics) DeliveryDuration(provider string, duration time.Duration) {
	m.deliveryDuration.WithLabelValues(provider).Observe(duration.Seconds())
}

var (
	_ coremetrics.Recorder = (*Metrics)(nil)
	_ coremetrics.Exporter = (*Metrics)(nil)
)
