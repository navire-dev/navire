package heartbeat

import (
	"fmt"
	"time"
)

type Ack struct {
	ID           string    `json:"id"`
	HeartbeatID  string    `json:"heartbeat_id"`
	DispatcherID string    `json:"dispatcher_id"`
	CoreID       string    `json:"core_id"`
	OccurredAt   time.Time `json:"occurred_at"`
	Accepted     bool      `json:"accepted"`
	Reason       string    `json:"reason,omitempty"`
}

func (a Ack) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("heartbeat acknowledgement ID is required")
	}
	if a.HeartbeatID == "" {
		return fmt.Errorf("heartbeat acknowledgement heartbeat ID is required")
	}
	if a.DispatcherID == "" {
		return fmt.Errorf("heartbeat acknowledgement dispatcher ID is required")
	}
	if a.CoreID == "" {
		return fmt.Errorf("heartbeat acknowledgement Core ID is required")
	}
	if a.OccurredAt.IsZero() {
		return fmt.Errorf("heartbeat acknowledgement timestamp is required")
	}
	if !a.Accepted && a.Reason == "" {
		return fmt.Errorf("rejected heartbeat acknowledgement reason is required")
	}
	return nil
}
