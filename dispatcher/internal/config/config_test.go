package config

import "testing"

func setRequiredRedis(t *testing.T) {
	t.Helper()
	t.Setenv("NAVIRE_DISPATCH_PLAN_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("NAVIRE_DSPC_REGISTRATION_SECRET", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("REDIS_ADDR", "localhost")
	t.Setenv("REDIS_PORT", "6379")
	t.Setenv("REDIS_USER", "dev")
	t.Setenv("REDIS_PASSWORD", "password")
}

func TestLoad(t *testing.T) {
	setRequiredRedis(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ConsumerID == "" {
		t.Fatal("ConsumerID is empty")
	}
}
