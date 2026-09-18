package template

import (
	"strings"
	"testing"

	"github.com/navire-dev/navire/shared/models"
	"gopkg.in/yaml.v3"
)

func TestValidateDataRejectsMissingFields(t *testing.T) {
	t.Parallel()

	definition := Definition{
		Key: "test",
		Variants: map[string]Variant{
			"failed": {
				Title: "{{ .service }} failed",
				Body:  "Request {{ .request_id }} requires attention.",
			},
		},
	}

	err := definition.ValidateData("failed", map[string]any{"service": "booking-api"})
	if err == nil || !strings.Contains(err.Error(), "request_id") {
		t.Fatalf("ValidateData() error = %v, want missing request_id", err)
	}
}

func TestValidateDataAcceptsAllReferencedFields(t *testing.T) {
	t.Parallel()

	definition := Definition{
		Key: "test",
		Variants: map[string]Variant{
			"failed": {
				Title: "{{ .service }} failed",
				Body:  "Request {{ .request_id }} requires attention.",
			},
		},
	}

	if err := definition.ValidateData("failed", map[string]any{
		"service":    "booking-api",
		"request_id": "req-123",
	}); err != nil {
		t.Fatalf("ValidateData() error = %v", err)
	}
}

func TestValidateRejectsUnsupportedDataExpression(t *testing.T) {
	t.Parallel()

	definition := Definition{
		Key: "test",
		Variants: map[string]Variant{
			"failed": {
				State: EventStateFailure,
				Title: "Failure",
				Body:  "{{ range .items }}{{ . }}{{ end }}",
			},
		},
		Providers: ProviderCatalog{Entries: map[string]ProviderDefinition{
			"gotify": {Endpoints: map[string]ProviderOverride{"test": {}}},
		}},
	}

	if err := definition.Validate(); err == nil || !strings.Contains(err.Error(), "top-level data fields") {
		t.Fatalf("Validate() error = %v, want unsupported expression error", err)
	}
}

func TestResolvePriority(t *testing.T) {
	t.Parallel()

	definition := Definition{
		Variants: map[string]Variant{
			"success": {Priority: PriorityLow},
			"default": {},
		},
	}

	tests := []struct {
		name     string
		variant  string
		override string
		want     Priority
	}{
		{name: "request override", variant: "success", override: "critical", want: PriorityCritical},
		{name: "variant priority", variant: "success", want: PriorityLow},
		{name: "template default", variant: "default", want: PriorityNormal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := definition.ResolvePriority(tt.variant, tt.override)
			if err != nil {
				t.Fatalf("ResolvePriority() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ResolvePriority() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveVariantByStatus(t *testing.T) {
	definition := Definition{Variants: map[string]Variant{
		"up":   {Status: "up"},
		"down": {Status: "down"},
	}}

	name, _, err := definition.ResolveVariant("", map[string]any{"status": "down"})
	if err != nil {
		t.Fatalf("ResolveVariant() error = %v", err)
	}
	if name != "down" {
		t.Fatalf("ResolveVariant() = %q, want down", name)
	}

	name, _, err = definition.ResolveVariant("up", map[string]any{"status": "down"})
	if err != nil {
		t.Fatalf("explicit ResolveVariant() error = %v", err)
	}
	if name != "up" {
		t.Fatalf("explicit ResolveVariant() = %q, want up", name)
	}
}

func TestResolveVariantByStatusRequiresMatch(t *testing.T) {
	definition := Definition{Variants: map[string]Variant{"up": {Status: "up"}}}
	if _, _, err := definition.ResolveVariant("", map[string]any{"status": "unknown"}); err == nil {
		t.Fatal("ResolveVariant() accepted an unknown status")
	}
}

func TestResolvePolicyMergesDefaultsAndProviderOverride(t *testing.T) {
	maxAttempts := 5
	backoff := models.BackoffLinear
	initialWait := "2s"

	definition := Definition{
		Providers: ProviderCatalog{
			Defaults: DeliveryPolicy{
				Timeout: "5s",
				Retry: RetryPolicy{
					MaxAttempts: 3,
					Backoff:     models.BackoffExponential,
					InitialWait: "1s",
				},
			},
			Entries: map[string]ProviderDefinition{
				"discord": {
					Endpoints: map[string]ProviderOverride{
						"test": {
							Timeout: func() *string { value := "3s"; return &value }(),
							Retry: RetryOverride{
								MaxAttempts: &maxAttempts,
								Backoff:     &backoff,
								InitialWait: &initialWait,
							},
						},
					},
				},
			},
		},
	}

	policy, err := definition.ResolvePolicy("discord", "test")
	if err != nil {
		t.Fatalf("ResolvePolicy() error = %v", err)
	}
	if policy.Timeout != "3s" || policy.Retry.MaxAttempts != 5 || policy.Retry.Backoff != models.BackoffLinear || policy.Retry.InitialWait != "2s" {
		t.Fatalf("ResolvePolicy() = %#v, want provider overrides", policy)
	}
}

func TestYAMLPolicyShapeLoads(t *testing.T) {
	var definition Definition
	if err := yaml.Unmarshal([]byte(`
key: cicd
variants:
  ok:
    title: ok
    body: ok
providers:
  defaults:
    timeout: 5s
    retry:
      max_attempts: 3
      backoff: exponential
      initial_wait: 1s
  discord:
    endpoints:
      test:
        timeout: 3s
        retry:
          max_attempts: 5
          backoff: linear
          initial_wait: 2s
`), &definition); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	policy, err := definition.ResolvePolicy("discord", "test")
	if err != nil {
		t.Fatalf("ResolvePolicy() error = %v", err)
	}
	if policy.Timeout != "3s" || policy.Retry.MaxAttempts != 5 || policy.Retry.Backoff != models.BackoffLinear {
		t.Fatalf("ResolvePolicy() = %#v, want YAML override", policy)
	}
}

func TestValidateRejectsInvalidPriority(t *testing.T) {
	t.Parallel()

	definition := Definition{
		Key: "test",
		Variants: map[string]Variant{
			"failed": {Priority: "urgent", Title: "Failed", Body: "Action required"},
		},
		Providers: ProviderCatalog{Entries: map[string]ProviderDefinition{
			"gotify": {Endpoints: map[string]ProviderOverride{"test": {}}},
		}},
	}
	if err := definition.Validate(); err == nil {
		t.Fatal("expected invalid priority to be rejected")
	}
}

func TestRoutingRulesLoadAndMergeIdenticalConditions(t *testing.T) {
	definition, err := Parse([]byte(`
key: routing
variants:
  failed:
    state: failure
    priority: critical
    title: failed
    body: failed
providers:
  defaults:
    timeout: 5s
    retry:
      max_attempts: 3
      backoff: exponential
      initial_wait: 5s
  gotify:
    endpoints:
      cicd: {}
      operations: {}
  ntfy:
    endpoints:
      cicd: {}
rules:
  - condition:
      priority: critical
      state: failure
      data:
        environment: production
    providers:
      gotify:
        endpoints:
          - cicd
  - condition:
      state: failure
      priority: critical
      data:
        environment: production
    providers:
      gotify:
        endpoints:
          - operations
      ntfy:
        endpoints:
          - cicd
`), "routing.yml")
	if err != nil {
		t.Fatalf("decodeDefinition() error = %v", err)
	}
	if err := definition.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(definition.Rules) != 1 {
		t.Fatalf("len(Rules) = %d, want 1", len(definition.Rules))
	}
	if got := definition.Rules[0].Providers["gotify"].Endpoints; len(got) != 2 || got[0] != "cicd" || got[1] != "operations" {
		t.Fatalf("gotify endpoints = %#v, want [cicd operations]", got)
	}
	if got := definition.Rules[0].Providers["ntfy"].Endpoints; len(got) != 1 || got[0] != "cicd" {
		t.Fatalf("ntfy endpoints = %#v, want [cicd]", got)
	}
}

func TestValidateRejectsRuleEndpointOutsideProviderCatalog(t *testing.T) {
	definition := Definition{
		Key: "routing",
		Variants: map[string]Variant{
			"failed": {State: EventStateFailure, Priority: PriorityCritical, Title: "failed", Body: "failed"},
		},
		Providers: ProviderCatalog{Entries: map[string]ProviderDefinition{
			"gotify": {Endpoints: map[string]ProviderOverride{"cicd": {}}},
		}},
		Rules: []RoutingRule{{
			Condition: RuleCondition{Priority: PriorityCritical},
			Providers: map[string]RuleProvider{
				"gotify": {Endpoints: []string{"missing"}},
			},
		}},
	}
	if err := definition.Validate(); err == nil || !strings.Contains(err.Error(), `endpoint "missing" is not defined`) {
		t.Fatalf("Validate() error = %v, want missing endpoint error", err)
	}
}
