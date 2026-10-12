package lifecycle

import (
	"testing"
	"time"
)

func TestEventValidate(t *testing.T) {
	event := Event{
		ID:         "core-1:shutdown:1",
		Kind:       CoreShutdown,
		OccurredAt: time.Now().UTC(),
		CoreID:     "core-1",
		Reason:     "signal",
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestEventValidateRejectsUnsupportedKind(t *testing.T) {
	event := Event{
		ID:         "core-1:event:1",
		Kind:       "unknown",
		OccurredAt: time.Now().UTC(),
		CoreID:     "core-1",
	}
	if err := event.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want unsupported kind error")
	}
}
