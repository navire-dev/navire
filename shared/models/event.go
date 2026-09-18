package models

import "fmt"

// EventState classifies the functional state represented by a template
// variant. It is independent from provider delivery success or failure.
type EventState string

const (
	EventStateSuccess       EventState = "success"
	EventStateFailure       EventState = "failure"
	EventStateInformational EventState = "informational"
)

func ParseEventState(value string) (EventState, error) {
	state := EventState(value)
	switch state {
	case EventStateSuccess, EventStateFailure, EventStateInformational:
		return state, nil
	default:
		return "", fmt.Errorf("invalid event state %q (must be success, failure, or informational)", value)
	}
}
