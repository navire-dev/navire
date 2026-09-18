package models

import "testing"

func TestParseEventState(t *testing.T) {
	for _, state := range []EventState{EventStateSuccess, EventStateFailure, EventStateInformational} {
		if got, err := ParseEventState(string(state)); err != nil || got != state {
			t.Fatalf("ParseEventState(%q) = %q, %v", state, got, err)
		}
	}
}

func TestParseEventStateRejectsUnknownState(t *testing.T) {
	if _, err := ParseEventState("warning"); err == nil {
		t.Fatal("ParseEventState accepted an unknown state")
	}
}
