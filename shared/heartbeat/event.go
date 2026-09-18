package heartbeat

import (
	"fmt"
	"time"
)

type Status string

const (
	Ready Status = "ready"
	Down  Status = "down"
)

type Event struct {
	ID                string    `json:"id"`
	OccurredAt        time.Time `json:"occurred_at"`
	DispatcherID      string    `json:"dispatcher_id"`
	RegistrationToken string    `json:"registration_token,omitempty"`
	Status            Status    `json:"status,omitempty"`
	RedisConnected    bool      `json:"redis_connected,omitempty"`
	StreamLag         int64     `json:"stream_lag,omitempty"`
}

func (e Event) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("heartbeat event ID is required")
	}
	if e.DispatcherID == "" {
		return fmt.Errorf("heartbeat dispatcher ID is required")
	}
	if e.Status != Ready && e.Status != Down {
		return fmt.Errorf("invalid heartbeat status %q", e.Status)
	}
	return nil
}
