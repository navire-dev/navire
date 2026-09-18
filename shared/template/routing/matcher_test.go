package routing

import (
	"reflect"
	"testing"

	messageTemplate "github.com/navire-dev/navire/shared/template"
)

func TestResolveUsesMostSpecificMatchingRules(t *testing.T) {
	rules := []messageTemplate.RoutingRule{
		{
			Condition: messageTemplate.RuleCondition{Priority: messageTemplate.PriorityCritical},
			Providers: map[string]messageTemplate.RuleProvider{
				"gotify": {Endpoints: []string{"cicd"}},
			},
		},
		{
			Condition: messageTemplate.RuleCondition{
				Priority: messageTemplate.PriorityCritical,
				State:    messageTemplate.EventStateFailure,
				Data:     map[string]any{"environment": "production"},
			},
			Providers: map[string]messageTemplate.RuleProvider{
				"ntfy": {Endpoints: []string{"cicd"}},
			},
		},
	}
	all := []Destination{
		{Provider: "gotify", Endpoint: "cicd"},
		{Provider: "ntfy", Endpoint: "cicd"},
	}

	got := Resolve(rules, Event{
		Priority: messageTemplate.PriorityCritical,
		State:    messageTemplate.EventStateFailure,
		Data:     map[string]any{"environment": "production"},
	}, all)
	want := []Destination{{Provider: "ntfy", Endpoint: "cicd"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Resolve() = %#v, want %#v", got, want)
	}
}

func TestResolveCombinesRulesAtSameSpecificity(t *testing.T) {
	rules := []messageTemplate.RoutingRule{
		{
			Condition: messageTemplate.RuleCondition{Priority: messageTemplate.PriorityCritical},
			Providers: map[string]messageTemplate.RuleProvider{
				"gotify": {Endpoints: []string{"cicd"}},
			},
		},
		{
			Condition: messageTemplate.RuleCondition{State: messageTemplate.EventStateFailure},
			Providers: map[string]messageTemplate.RuleProvider{
				"ntfy": {Endpoints: []string{"cicd"}},
			},
		},
	}

	got := Resolve(rules, Event{
		Priority: messageTemplate.PriorityCritical,
		State:    messageTemplate.EventStateFailure,
	}, nil)
	want := []Destination{
		{Provider: "gotify", Endpoint: "cicd"},
		{Provider: "ntfy", Endpoint: "cicd"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Resolve() = %#v, want %#v", got, want)
	}
}

func TestResolveFallsBackToAllDestinations(t *testing.T) {
	all := []Destination{
		{Provider: "ntfy", Endpoint: "cicd"},
		{Provider: "gotify", Endpoint: "cicd"},
		{Provider: "gotify", Endpoint: "cicd"},
	}
	rules := []messageTemplate.RoutingRule{{
		Condition: messageTemplate.RuleCondition{Priority: messageTemplate.PriorityCritical},
		Providers: map[string]messageTemplate.RuleProvider{
			"ntfy": {Endpoints: []string{"cicd"}},
		},
	}}

	got := Resolve(rules, Event{Priority: messageTemplate.PriorityNormal}, all)
	want := []Destination{
		{Provider: "gotify", Endpoint: "cicd"},
		{Provider: "ntfy", Endpoint: "cicd"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Resolve() = %#v, want %#v", got, want)
	}
}

func TestResolveCatchAllLosesToMoreSpecificRule(t *testing.T) {
	rules := []messageTemplate.RoutingRule{
		{
			Condition: messageTemplate.RuleCondition{},
			Providers: map[string]messageTemplate.RuleProvider{
				"gotify": {Endpoints: []string{"operations"}},
			},
		},
		{
			Condition: messageTemplate.RuleCondition{Priority: messageTemplate.PriorityCritical},
			Providers: map[string]messageTemplate.RuleProvider{
				"ntfy": {Endpoints: []string{"cicd"}},
			},
		},
	}

	got := Resolve(rules, Event{Priority: messageTemplate.PriorityCritical}, nil)
	want := []Destination{{Provider: "ntfy", Endpoint: "cicd"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Resolve() = %#v, want %#v", got, want)
	}
}
