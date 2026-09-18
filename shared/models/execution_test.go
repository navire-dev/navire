package models

import (
	"encoding/json"
	"testing"

	sharedproviders "github.com/navire-dev/navire/shared/providers"
)

func TestExecutionPlanContainsOneTarget(t *testing.T) {
	plan := ExecutionPlan{
		SchemaVersion:  ExecutionSchemaVersion,
		MessageID:      "event-1",
		IdempotencyKey: "event-1:gotify_prod",
		Target: ExecutionTarget{
			Name:     "gotify_prod",
			Provider: sharedproviders.Gotify,
		},
	}
	payload, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded ExecutionPlan
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded.Target.Name != plan.Target.Name || decoded.Target.Provider != plan.Target.Provider {
		t.Fatalf("decoded target = %#v, want %#v", decoded.Target, plan.Target)
	}
}
