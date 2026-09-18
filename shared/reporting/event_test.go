package reporting

import (
	"testing"
	"time"

	sharedproviders "github.com/navire-dev/navire/shared/providers"
)

func TestEventValidate(t *testing.T) {
	t.Parallel()

	valid := Event{
		ID:           "delivery-1",
		Kind:         KindDeliveryResult,
		OccurredAt:   time.Now(),
		DispatcherID: "dspc-1",
		MessageID:    "message-1",
		TargetName:   "gotify_prod",
		Provider:     sharedproviders.Gotify,
		State:        DeliverySuccess,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	invalid := valid
	invalid.State = "unknown"
	if err := invalid.Validate(); err == nil {
		t.Fatal("Validate() accepted an invalid delivery state")
	}
}
