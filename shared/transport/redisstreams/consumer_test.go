package redisstreams

import (
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestValidateConsumerGroupConfig(t *testing.T) {
	tests := []struct {
		name     string
		client   *redis.Client
		stream   string
		group    string
		consumer string
		want     string
	}{
		{name: "missing client", stream: "stream", group: "group", consumer: "consumer", want: "redis client is required"},
		{name: "missing stream", client: redis.NewClient(&redis.Options{}), group: "group", consumer: "consumer", want: "stream is required"},
		{name: "missing group", client: redis.NewClient(&redis.Options{}), stream: "stream", consumer: "consumer", want: "consumer group is required"},
		{name: "missing consumer", client: redis.NewClient(&redis.Options{}), stream: "stream", group: "group", want: "consumer name is required"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateConsumerGroupConfig(test.client, test.stream, test.group, test.consumer); err == nil || err.Error() != test.want {
				t.Fatalf("validateConsumerGroupConfig() error = %v, want %q", err, test.want)
			}
			if test.client != nil {
				_ = test.client.Close()
			}
		})
	}
}

func TestFlattenConsumerMessages(t *testing.T) {
	entries := []redis.XStream{
		{Stream: "one", Messages: []redis.XMessage{{ID: "1-0"}}},
		{Stream: "two", Messages: []redis.XMessage{{ID: "2-0"}, {ID: "2-1"}}},
	}

	messages := flattenConsumerMessages(entries)
	if len(messages) != 3 {
		t.Fatalf("flattenConsumerMessages() returned %d messages, want 3", len(messages))
	}
	for i, want := range []string{"1-0", "2-0", "2-1"} {
		if messages[i].ID != want {
			t.Fatalf("message %d ID = %q, want %q", i, messages[i].ID, want)
		}
	}
}
