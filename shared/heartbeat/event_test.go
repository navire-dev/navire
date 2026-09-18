package heartbeat

import (
	"testing"
	"time"
)

func TestEventValidate(t *testing.T) {
	event := Event{ID: "dspc-1:heartbeat:1", DispatcherID: "dspc-1", Status: Ready, OccurredAt: time.Now().UTC()}
	if err := event.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestAckValidateRequiresReasonWhenRejected(t *testing.T) {
	ack := Ack{ID: "core-1:heartbeat-ack:1", HeartbeatID: "heartbeat-1", DispatcherID: "dspc-1", CoreID: "core-1", OccurredAt: time.Now().UTC()}
	if err := ack.Validate(); err == nil {
		t.Fatal("Validate() accepted a rejected acknowledgement without a reason")
	}
}
