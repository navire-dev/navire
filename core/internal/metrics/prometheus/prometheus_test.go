package prometheus

import (
	"strings"
	"testing"
	"time"

	"github.com/navire-dev/navire/shared/providers"
)

func TestScrapeAggregatesCoreAndLocalDeliveryMetrics(t *testing.T) {
	m := New(time.Now().Add(-time.Second))
	m.EventReceived()
	m.EventAccepted()
	m.Event("deployments", "failed", "failure")
	m.EventPublished()
	m.StreamPublished(1)
	m.StreamConsumed(string(providers.Gotify))
	m.StreamAcked(string(providers.Gotify))
	m.DeliveryAttempt(string(providers.Gotify), "success")
	m.DeliveryDuration(string(providers.Gotify), 20*time.Millisecond)

	payload, err := m.Scrape(t.Context())
	if err != nil {
		t.Fatalf("Scrape() error = %v", err)
	}
	body := string(payload.Body)
	for _, metric := range []string{
		`navire_events_total{component="core",state="failure",template="deployments",variant="failed"} 1`,
		`navire_stream_messages_consumed_total{component="core",provider="gotify"} 1`,
		`navire_stream_messages_acked_total{component="core",provider="gotify"} 1`,
		`navire_delivery_attempts_total{component="core",provider="gotify",state="success"} 1`,
	} {
		if !strings.Contains(body, metric) {
			t.Fatalf("Scrape() missing %q\n%s", metric, body)
		}
	}
	if strings.Contains(body, "navire_dispatcher_") {
		t.Fatalf("Scrape() exposed stale dispatcher metrics\n%s", body)
	}
}
