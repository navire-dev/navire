package prometheus

import (
	"strings"
	"testing"
	"time"

	"github.com/navire-dev/navire/shared/heartbeat"
	"github.com/navire-dev/navire/shared/providers"
	"github.com/navire-dev/navire/shared/reporting"
)

func TestScrapeAggregatesCoreAndDispatcherReporting(t *testing.T) {
	m := New(time.Now().Add(-time.Second))
	m.EventReceived()
	m.EventAccepted()
	m.Event("deployments", "failed", "failure")
	m.EventPublished()
	m.StreamPublished(1)
	m.ObserveReporting(reporting.Event{
		ID:           "delivery-result-1",
		Kind:         reporting.KindDeliveryResult,
		DispatcherID: "dspc-1",
		Provider:     providers.Gotify,
		State:        reporting.DeliverySuccess,
		DurationMS:   20,
	})
	m.ObserveReporting(reporting.Event{
		ID:           "delivery-result-1",
		Kind:         reporting.KindDeliveryResult,
		DispatcherID: "dspc-1",
		Provider:     providers.Gotify,
		State:        reporting.DeliverySuccess,
		DurationMS:   20,
	})
	m.ObserveHeartbeat(heartbeat.Event{
		ID:             "heartbeat-1",
		DispatcherID:   "dspc-1",
		Status:         heartbeat.Ready,
		RedisConnected: true,
		StreamLag:      2,
		OccurredAt:     time.Now(),
	})

	payload, err := m.Scrape(t.Context())
	if err != nil {
		t.Fatalf("Scrape() error = %v", err)
	}
	body := string(payload.Body)
	for _, metric := range []string{
		`navire_events_total{component="core",state="failure",template="deployments",variant="failed"} 1`,
		`navire_delivery_attempts_total{component="core",provider="gotify",state="success"} 1`,
		`navire_dispatcher_up{component="core",dispatcher="dspc-1"} 1`,
		`navire_stream_consumer_lag{component="core",dispatcher="dspc-1"} 2`,
	} {
		if !strings.Contains(body, metric) {
			t.Fatalf("Scrape() missing %q\n%s", metric, body)
		}
	}
}

func TestScrapeRemovesStaleDispatcherMetrics(t *testing.T) {
	m := New(time.Now())
	m.ObserveHeartbeat(heartbeat.Event{
		ID:             "heartbeat-stale",
		DispatcherID:   "dspc-stale",
		Status:         heartbeat.Ready,
		RedisConnected: true,
		StreamLag:      2,
		OccurredAt:     time.Now().Add(-time.Minute),
	})

	payload, err := m.Scrape(t.Context())
	if err != nil {
		t.Fatalf("Scrape() error = %v", err)
	}
	body := string(payload.Body)
	for _, metric := range []string{
		`dispatcher="dspc-stale"`,
	} {
		if strings.Contains(body, metric) {
			t.Fatalf("Scrape() kept stale dispatcher metric %q\n%s", metric, body)
		}
	}
}
