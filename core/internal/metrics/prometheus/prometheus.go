package prometheus

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	coremetrics "github.com/navire-dev/navire/core/internal/metrics"
	"github.com/navire-dev/navire/shared/heartbeat"
	"github.com/navire-dev/navire/shared/reporting"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/common/expfmt"
)

// Metrics is the Prometheus implementation of the Core metrics contract.
// It aggregates local Core facts and reporting received from DSPC instances.
type Metrics struct {
	registry *prometheus.Registry
	started  time.Time
	mu       sync.Mutex

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
	streamConsumed     prometheus.Counter
	streamAcked        prometheus.Counter
	deliveryAttempts   *prometheus.CounterVec
	deliveryDuration   *prometheus.HistogramVec
	dispatcherUp       *prometheus.GaugeVec
	dispatcherRedis    *prometheus.GaugeVec
	streamLag          *prometheus.GaugeVec
	lastHeartbeat      map[string]time.Time
	seenReporting      map[string]time.Time
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
		streamConsumed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "navire_stream_messages_consumed_total", Help: "Notification plans consumed by DSPC.", ConstLabels: labels,
		}),
		streamAcked: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "navire_stream_messages_acked_total", Help: "Notification plans acknowledged by DSPC.", ConstLabels: labels,
		}),
		deliveryAttempts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "navire_delivery_attempts_total", Help: "Delivery attempts by provider and state.", ConstLabels: labels,
		}, []string{"provider", "state"}),
		deliveryDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "navire_delivery_duration_seconds", Help: "Delivery duration by provider.", ConstLabels: labels,
		}, []string{"provider"}),
		dispatcherUp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "navire_dispatcher_up", Help: "Whether a Dispatcher heartbeat is fresh.", ConstLabels: labels,
		}, []string{"dispatcher"}),
		dispatcherRedis: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "navire_dispatcher_redis_connected", Help: "Whether a Dispatcher reports Redis connected.", ConstLabels: labels,
		}, []string{"dispatcher"}),
		streamLag: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "navire_stream_consumer_lag", Help: "Dispatcher Redis Stream consumer lag.", ConstLabels: labels,
		}, []string{"dispatcher"}),
		lastHeartbeat: make(map[string]time.Time),
		seenReporting: make(map[string]time.Time),
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
		m.dispatcherUp,
		m.dispatcherRedis,
		m.streamLag,
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)
	return m
}

func (m *Metrics) Scrape(_ context.Context) (coremetrics.Payload, error) {
	m.markStaleHeartbeats()
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
	m.mu.Lock()
	defer m.mu.Unlock()
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

func (m *Metrics) ObserveReporting(event reporting.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, seen := m.seenReporting[event.ID]; seen {
		return
	}
	m.seenReporting[event.ID] = time.Now()
	if len(m.seenReporting) > 100000 {
		m.pruneReporting()
	}
	switch event.Kind {
	case reporting.KindDeliveryResult:
		m.streamConsumed.Inc()
		m.deliveryAttempts.WithLabelValues(string(event.Provider), string(event.State)).Inc()
		m.deliveryDuration.WithLabelValues(string(event.Provider)).Observe(float64(event.DurationMS) / 1000)
	case reporting.KindDeliveryAck:
		m.streamAcked.Inc()
	}
}

func (m *Metrics) ObserveHeartbeat(event heartbeat.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastHeartbeat[event.DispatcherID] = event.OccurredAt
	m.dispatcherUp.WithLabelValues(event.DispatcherID).Set(1)
	if event.RedisConnected {
		m.dispatcherRedis.WithLabelValues(event.DispatcherID).Set(1)
	} else {
		m.dispatcherRedis.WithLabelValues(event.DispatcherID).Set(0)
	}
	m.streamLag.WithLabelValues(event.DispatcherID).Set(float64(event.StreamLag))
}

func (m *Metrics) pruneReporting() {
	cutoff := time.Now().Add(-time.Hour)
	for id, seenAt := range m.seenReporting {
		if seenAt.Before(cutoff) {
			delete(m.seenReporting, id)
		}
	}
}

func (m *Metrics) markStaleHeartbeats() {
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff := time.Now().Add(-30 * time.Second)
	for dispatcher, last := range m.lastHeartbeat {
		if last.Before(cutoff) {
			m.dispatcherUp.DeleteLabelValues(dispatcher)
			m.dispatcherRedis.DeleteLabelValues(dispatcher)
			m.streamLag.DeleteLabelValues(dispatcher)
			delete(m.lastHeartbeat, dispatcher)
		}
	}
}

var (
	_ coremetrics.Recorder = (*Metrics)(nil)
	_ coremetrics.Exporter = (*Metrics)(nil)
)
