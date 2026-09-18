package lifecycle

import (
	"fmt"
	"time"
)

// Kind identifies a lifecycle event exchanged by Navire components.
type Kind string

const CoreShutdown Kind = "core_shutdown"

// Event is the transport-independent lifecycle contract.
type Event struct {
	ID         string    `json:"id"`
	Kind       Kind      `json:"kind"`
	OccurredAt time.Time `json:"occurred_at"`
	CoreID     string    `json:"core_id"`
	Reason     string    `json:"reason,omitempty"`
}

func (e Event) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("lifecycle event ID is required")
	}
	if e.Kind != CoreShutdown {
		return fmt.Errorf("unsupported lifecycle event kind %q", e.Kind)
	}
	if e.OccurredAt.IsZero() {
		return fmt.Errorf("lifecycle event occurrence time is required")
	}
	if e.CoreID == "" {
		return fmt.Errorf("lifecycle event Core ID is required")
	}
	return nil
}
