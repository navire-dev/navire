package models

import (
	"fmt"
	"time"

	sharedproviders "github.com/navire-dev/navire/shared/providers"
)

const ExecutionSchemaVersion = 2

// ExecutionPlan is the immutable snapshot published by Core and consumed by
// a Dispatcher. Each plan contains exactly one delivery target.
type ExecutionPlan struct {
	SchemaVersion      int              `json:"schema_version"`
	TransportMessageID string           `json:"-"`
	MessageID          string           `json:"message_id"` // parent notification ID
	TenantID           string           `json:"tenant_id,omitempty"`
	IdempotencyKey     string           `json:"idempotency_key"` // target-specific delivery ID
	CreatedAt          time.Time        `json:"created_at"`
	ExpiresAt          *time.Time       `json:"expires_at,omitempty"`
	Variant            ExecutionVariant `json:"variant"`
	Data               map[string]any   `json:"data,omitempty"`
	Target             ExecutionTarget  `json:"target"`
}

type ExecutionVariant struct {
	Name     string     `json:"name"`
	Title    string     `json:"title"`
	Body     string     `json:"body"`
	Priority Priority   `json:"priority"`
	State    EventState `json:"state"`
}

type ExecutionTarget struct {
	Name     string             `json:"name"`
	Provider sharedproviders.ID `json:"provider"`
	URL      string             `json:"url"` // encrypted complete URL
	Policy   DeliveryPolicy     `json:"policy"`
}

type DeliveryPolicy struct {
	Timeout string      `yaml:"timeout" json:"timeout"`
	Retry   RetryPolicy `yaml:"retry" json:"retry"`
}

type RetryPolicy struct {
	MaxAttempts int             `yaml:"max_attempts" json:"max_attempts"`
	Backoff     BackoffStrategy `yaml:"backoff" json:"backoff"`
	InitialWait string          `yaml:"initial_wait" json:"initial_wait"`
}

type BackoffStrategy string

const (
	BackoffNone        BackoffStrategy = "none"
	BackoffLinear      BackoffStrategy = "linear"
	BackoffExponential BackoffStrategy = "exponential"
)

func (p DeliveryPolicy) Validate() error {
	if p.Timeout == "" {
		return fmt.Errorf("timeout is required")
	}
	timeout, err := time.ParseDuration(p.Timeout)
	if err != nil || timeout <= 0 {
		return fmt.Errorf("timeout must be a positive duration")
	}
	if p.Retry.MaxAttempts < 1 {
		return fmt.Errorf("retry.max_attempts must be at least 1")
	}
	switch p.Retry.Backoff {
	case BackoffNone, BackoffLinear, BackoffExponential:
	default:
		return fmt.Errorf("retry.backoff must be none, linear, or exponential")
	}
	if p.Retry.MaxAttempts > 100 {
		return fmt.Errorf("retry.max_attempts must be at most 100")
	}
	if p.Retry.MaxAttempts > 1 && p.Retry.Backoff != BackoffNone {
		if p.Retry.InitialWait == "" {
			return fmt.Errorf("retry.initial_wait is required when max_attempts is greater than 1")
		}
		wait, err := time.ParseDuration(p.Retry.InitialWait)
		if err != nil || wait <= 0 {
			return fmt.Errorf("retry.initial_wait must be a positive duration")
		}
	}
	return nil
}
