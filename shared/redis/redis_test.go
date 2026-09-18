package redis

import (
	"strings"
	"testing"
	"time"

	sharedconfig "github.com/navire-dev/navire/shared/config"
)

func validConfig() Config {
	return Config{
		Addr:          "localhost:6379",
		User:          "user",
		Password:      "password",
		DB:            0,
		DialTimeout:   5 * time.Second,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		PoolSize:      10,
		RetryInterval: 2 * time.Second,
		MaxWait:       10 * time.Second,
		PingTimeout:   5 * time.Second,
		WarnThreshold: 3,
	}
}

func TestConfigValidate(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestConfigValidateURI(t *testing.T) {
	cfg := validConfig()
	cfg.URI = "rediss://user:password@redis.example.com:6380/2"
	cfg.Addr = ""
	cfg.User = ""
	cfg.Password = ""
	cfg.DB = 2
	cfg.TLSEnabled = true

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestLoadRejectsMixedURIAndSplitConfiguration(t *testing.T) {
	t.Setenv("REDIS_URI", "rediss://redis.example.com:6380/0")
	t.Setenv("REDIS_PASSWORD", "password")

	loader := sharedconfig.NewLoader()
	Load(loader)

	err := loader.Err()
	if err == nil || !strings.Contains(err.Error(), "REDIS_URI cannot be combined") {
		t.Fatalf("Load() error = %v, want mixed configuration error", err)
	}
}

func TestLoadAllowsEmptySplitCredentials(t *testing.T) {
	t.Setenv("REDIS_URI", "")
	t.Setenv("REDIS_ADDR", "localhost")
	t.Setenv("REDIS_PORT", "6379")
	t.Setenv("REDIS_USER", "")
	t.Setenv("REDIS_PASSWORD", "")

	loader := sharedconfig.NewLoader()
	cfg := Load(loader)
	if err := loader.Err(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.User != "" || cfg.Password != "" {
		t.Fatalf("credentials = %q/%q, want empty credentials", cfg.User, cfg.Password)
	}
}

func TestConfigValidateRejectsInsecureTLSWithoutTLS(t *testing.T) {
	cfg := validConfig()
	cfg.TLSInsecureSkipVerify = true

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "REDIS_TLS_INSECURE_SKIP_VERIFY") {
		t.Fatalf("Validate() error = %v, want TLS dependency error", err)
	}
}

func TestConfigValidateRejectsInvalidRetryInterval(t *testing.T) {
	cfg := validConfig()
	cfg.RetryInterval = 20 * time.Second
	cfg.MaxWait = 10 * time.Second

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "REDIS_RETRY_INTERVAL") {
		t.Fatalf("Validate() error = %v, want retry interval error", err)
	}
}
