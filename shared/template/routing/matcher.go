package routing

import (
	"reflect"
	"sort"

	messageTemplate "github.com/navire-dev/navire/shared/template"
)

type Event struct {
	Priority messageTemplate.Priority
	State    messageTemplate.EventState
	Data     map[string]any
}

type Destination struct {
	Provider string
	Endpoint string
}

func Resolve(rules []messageTemplate.RoutingRule, event Event, all []Destination) []Destination {
	if len(rules) == 0 {
		return uniqueSorted(all)
	}

	maxSpecificity := -1
	selected := make(map[Destination]struct{})
	for _, rule := range rules {
		if !matches(rule.Condition, event) {
			continue
		}
		specificity := ruleSpecificity(rule.Condition)
		if specificity > maxSpecificity {
			maxSpecificity = specificity
			selected = make(map[Destination]struct{})
		}
		if specificity != maxSpecificity {
			continue
		}
		addDestinations(selected, rule)
	}

	if maxSpecificity < 0 {
		return uniqueSorted(all)
	}
	return sortedDestinations(selected)
}

func matches(condition messageTemplate.RuleCondition, event Event) bool {
	if condition.Priority != "" && condition.Priority != event.Priority {
		return false
	}
	if condition.State != "" && condition.State != event.State {
		return false
	}
	for key, expected := range condition.Data {
		actual, ok := event.Data[key]
		if !ok || !reflect.DeepEqual(expected, actual) {
			return false
		}
	}
	return true
}

func ruleSpecificity(condition messageTemplate.RuleCondition) int {
	specificity := len(condition.Data)
	if condition.Priority != "" {
		specificity++
	}
	if condition.State != "" {
		specificity++
	}
	return specificity
}

func addDestinations(selected map[Destination]struct{}, rule messageTemplate.RoutingRule) {
	for provider, destination := range rule.Providers {
		for _, endpoint := range destination.Endpoints {
			selected[Destination{Provider: provider, Endpoint: endpoint}] = struct{}{}
		}
	}
}

func uniqueSorted(destinations []Destination) []Destination {
	selected := make(map[Destination]struct{}, len(destinations))
	for _, destination := range destinations {
		selected[destination] = struct{}{}
	}
	return sortedDestinations(selected)
}

func sortedDestinations(selected map[Destination]struct{}) []Destination {
	result := make([]Destination, 0, len(selected))
	for destination := range selected {
		result = append(result, destination)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Provider == result[j].Provider {
			return result[i].Endpoint < result[j].Endpoint
		}
		return result[i].Provider < result[j].Provider
	})
	return result
}
