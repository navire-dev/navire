package template

import (
	"fmt"

	"github.com/navire-dev/navire/shared/models"
)

type (
	Priority       = models.Priority
	EventState     = models.EventState
	DeliveryPolicy = models.DeliveryPolicy
	RetryPolicy    = models.RetryPolicy
)

const (
	PriorityLow      = models.PriorityLow
	PriorityNormal   = models.PriorityNormal
	PriorityHigh     = models.PriorityHigh
	PriorityCritical = models.PriorityCritical
)

const (
	EventStateSuccess       = models.EventStateSuccess
	EventStateFailure       = models.EventStateFailure
	EventStateInformational = models.EventStateInformational
)

// Definition is the subset of the template contract needed by the MVP path.
// Delivery overrides remain in their YAML form until the complete model lands.
type Definition struct {
	TenantID    string             `yaml:"tenantID" json:"tenantID"`
	Key         string             `yaml:"key" json:"key"`
	Name        string             `yaml:"name" json:"name"`
	Description string             `yaml:"description" json:"description"`
	Variants    map[string]Variant `yaml:"variants" json:"variants"`
	Providers   ProviderCatalog    `yaml:"providers" json:"providers"`
	Rules       []RoutingRule      `yaml:"rules,omitempty" json:"rules,omitempty"`
}

type Variant struct {
	Status         string     `yaml:"status,omitempty" json:"status,omitempty"`
	Priority       Priority   `yaml:"priority" json:"priority"`
	State          EventState `yaml:"state" json:"state"`
	Title          string     `yaml:"title" json:"title"`
	Body           string     `yaml:"body" json:"body"`
	RequiredFields []string   `yaml:"-" json:"-"`
}

type ProviderCatalog struct {
	Defaults DeliveryPolicy                `yaml:"defaults" json:"defaults"`
	Entries  map[string]ProviderDefinition `yaml:",inline" json:",inline"`
}

type ProviderDefinition struct {
	Endpoints map[string]ProviderOverride `yaml:"endpoints" json:"endpoints"`
}

type ProviderOverride struct {
	Timeout *string       `yaml:"timeout" json:"timeout"`
	Retry   RetryOverride `yaml:"retry" json:"retry"`
}

type RoutingRule struct {
	Condition RuleCondition           `yaml:"condition" json:"condition"`
	Providers map[string]RuleProvider `yaml:"providers" json:"providers"`
}

type RuleCondition struct {
	Priority Priority       `yaml:"priority,omitempty" json:"priority,omitempty"`
	State    EventState     `yaml:"state,omitempty" json:"state,omitempty"`
	Data     map[string]any `yaml:"data,omitempty" json:"data,omitempty"`
}

type RuleProvider struct {
	Endpoints []string `yaml:"endpoints" json:"endpoints"`
}

type RetryOverride struct {
	MaxAttempts *int                    `yaml:"max_attempts" json:"max_attempts"`
	Backoff     *models.BackoffStrategy `yaml:"backoff" json:"backoff"`
	InitialWait *string                 `yaml:"initial_wait" json:"initial_wait"`
}

func ParsePriority(value string) (Priority, error) {
	priority := Priority(value)
	if priority == "" {
		return "", nil
	}
	switch priority {
	case PriorityLow, PriorityNormal, PriorityHigh, PriorityCritical:
		return priority, nil
	default:
		return "", fmt.Errorf("invalid priority %q (must be low, normal, high, or critical)", value)
	}
}
