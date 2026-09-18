package redisstreams

import (
	"context"
	"testing"

	"github.com/navire-dev/navire/shared/heartbeat"
)

func TestHeartbeatAckStreamIsScopedToDispatcher(t *testing.T) {
	if got, want := HeartbeatAckStream("dspc-1"), "navire:heartbeat-ack:dspc-1"; got != want {
		t.Fatalf("HeartbeatAckStream() = %q, want %q", got, want)
	}
	if got, want := HeartbeatAckGroup("dspc-1"), "navire-dspc-heartbeat-ack:dspc-1"; got != want {
		t.Fatalf("HeartbeatAckGroup() = %q, want %q", got, want)
	}
}

func TestHeartbeatAckConsumerRejectsInvalidConfiguration(t *testing.T) {
	consumer := HeartbeatAckConsumer{}
	if err := consumer.Run(t.Context(), func(context.Context, heartbeat.Ack) error { return nil }); err == nil {
		t.Fatal("Run() accepted a nil Redis client")
	}
}
