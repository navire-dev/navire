package reporting

import (
	"fmt"
	"time"

	sharedproviders "github.com/navire-dev/navire/shared/providers"
)

// Kind identifies a reporting event emitted by a Dispatcher and consumed by
// Core. The transport carrying these events is deliberately not part of this
// contract.
type Kind string

const (
	KindDeliveryResult Kind = "delivery_result"
	KindDeliveryAck    Kind = "delivery_ack"
)

type DeliveryState string

const (
	DeliverySuccess DeliveryState = "success"
	DeliveryFailure DeliveryState = "failure"
	DeliverySkipped DeliveryState = "skipped"
)

// Event is the transport-independent DSPC -> Core reporting contract.
// Fields not relevant to a kind remain empty.
type Event struct {
	ID                string             `json:"id"`
	Kind              Kind               `json:"kind"`
	OccurredAt        time.Time          `json:"occurred_at"`
	DispatcherID      string             `json:"dispatcher_id"`
	RegistrationToken string             `json:"registration_token,omitempty"`
	MessageID         string             `json:"message_id,omitempty"`
	TargetName        string             `json:"target_name,omitempty"`
	Provider          sharedproviders.ID `json:"provider,omitempty"`
	State             DeliveryState      `json:"state,omitempty"`
	DurationMS        int64              `json:"duration_ms,omitempty"`
	ErrorType         string             `json:"error_type,omitempty"`
}

func (e Event) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("reporting event ID is required")
	}
	if e.Kind == "" {
		return fmt.Errorf("reporting event kind is required")
	}
	if e.DispatcherID == "" {
		return fmt.Errorf("reporting dispatcher ID is required")
	}
	switch e.Kind {
	case KindDeliveryResult:
		return e.validateDeliveryResult()
	case KindDeliveryAck:
		return e.validateDeliveryAck()
	default:
		return fmt.Errorf("unsupported reporting event kind %q", e.Kind)
	}
}

func (e Event) validateDeliveryResult() error {
	if e.MessageID == "" || e.TargetName == "" || e.Provider == "" {
		return fmt.Errorf("delivery result requires message ID, target name and provider")
	}
	if e.State != DeliverySuccess && e.State != DeliveryFailure && e.State != DeliverySkipped {
		return fmt.Errorf("invalid delivery state %q", e.State)
	}
	return nil
}

func (e Event) validateDeliveryAck() error {
	if e.MessageID == "" || e.TargetName == "" {
		return fmt.Errorf("delivery ack requires message ID and target name")
	}
	return nil
}
