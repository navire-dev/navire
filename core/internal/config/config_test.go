package config

import (
	"strings"
	"testing"
	"time"
)

func setRequiredDirectories(t *testing.T) {
	t.Helper()
	t.Setenv("NAVIRE_TARGETS_DIR", "../config/targets")
	t.Setenv("NAVIRE_TEMPLATES_DIR", "../config/templates")
	t.Setenv("NAVIRE_DISPATCH_PLAN_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("NAVIRE_SECRETS_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("NAVIRE_DSPC_REGISTRATION_SECRET", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("REDIS_ADDR", "localhost")
	t.Setenv("REDIS_PORT", "6379")
	t.Setenv("REDIS_USER", "dev")
	t.Setenv("REDIS_PASSWORD", "password")
}

func TestLoadUsesDefaultsAndWildcardAllowlist(t *testing.T) {
	setRequiredDirectories(t)
	t.Setenv("NAVIRE_EVENTS_ALLOWED_CIDRS", "")
	t.Setenv("NAVIRE_EVENTS_ALLOWED_PROXIES", "127.0.0.1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.AllowedCIDRS) != 1 || cfg.AllowedCIDRS[0] != "*" {
		t.Fatalf("AllowedCIDRS = %#v, want wildcard", cfg.AllowedCIDRS)
	}
	if cfg.PlanRetention != 30*24*time.Hour || cfg.PlanCleanupInterval != time.Hour {
		t.Fatalf("plan cleanup defaults = %s, %s", cfg.PlanRetention, cfg.PlanCleanupInterval)
	}
	if cfg.ReportingRetention != 24*time.Hour || cfg.ReportingCleanupInterval != time.Hour {
		t.Fatalf("reporting cleanup defaults = %s, %s", cfg.ReportingRetention, cfg.ReportingCleanupInterval)
	}
	if cfg.EventsRateLimit != nil {
		t.Fatal("EventsRateLimit is enabled without rate-limit variables")
	}
}

func TestLoadConfiguresOptionalEventRateLimit(t *testing.T) {
	setRequiredDirectories(t)
	t.Setenv("NAVIRE_EVENTS_ALLOWED_PROXIES", "127.0.0.1")
	t.Setenv("NAVIRE_EVENTS_RATE_LIMIT_BURST", "100")
	t.Setenv("NAVIRE_EVENTS_RATE_LIMIT_REFILL_PER_MIN", "60")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.EventsRateLimit == nil {
		t.Fatal("EventsRateLimit is nil")
	}
	if cfg.EventsRateLimit.Burst != 100 || cfg.EventsRateLimit.RefillPerIPPerMin != 60 {
		t.Fatalf("EventsRateLimit = %#v", cfg.EventsRateLimit)
	}
}

func TestLoadRejectsPartialEventRateLimit(t *testing.T) {
	setRequiredDirectories(t)
	t.Setenv("NAVIRE_EVENTS_ALLOWED_PROXIES", "127.0.0.1")
	t.Setenv("NAVIRE_EVENTS_RATE_LIMIT_BURST", "100")
	t.Setenv("NAVIRE_EVENTS_RATE_LIMIT_REFILL_PER_MIN", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "must be configured together") {
		t.Fatalf("Load() error = %v, want partial rate-limit configuration error", err)
	}
}

func TestLoadRejectsInvalidRedisDuration(t *testing.T) {
	setRequiredDirectories(t)
	t.Setenv("REDIS_PING_TIMEOUT", "3seconds")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "REDIS_PING_TIMEOUT") {
		t.Fatalf("Load() error = %v, want REDIS_PING_TIMEOUT error", err)
	}
}

func TestLoadRejectsInvalidCIDR(t *testing.T) {
	setRequiredDirectories(t)
	t.Setenv("NAVIRE_EVENTS_ALLOWED_PROXIES", "127.0.0.1")
	t.Setenv("NAVIRE_EVENTS_ALLOWED_CIDRS", "127.0.0.1,not-an-ip")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "not-an-ip") {
		t.Fatalf("Load() error = %v, want invalid CIDR error", err)
	}
}

func TestLoadUsesDefaultDirectories(t *testing.T) {
	t.Setenv("NAVIRE_DISPATCH_PLAN_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("NAVIRE_SECRETS_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("NAVIRE_DSPC_REGISTRATION_SECRET", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("REDIS_ADDR", "localhost")
	t.Setenv("REDIS_PORT", "6379")
	t.Setenv("REDIS_USER", "dev")
	t.Setenv("REDIS_PASSWORD", "password")
	t.Setenv("NAVIRE_TARGETS_DIR", "")
	t.Setenv("NAVIRE_TEMPLATES_DIR", "")
	t.Setenv("NAVIRE_EVENTS_ALLOWED_PROXIES", "127.0.0.1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.TargetsDir != "/config/targets" || cfg.TemplatesDir != "/config/templates" {
		t.Fatalf("default directories = %q, %q", cfg.TargetsDir, cfg.TemplatesDir)
	}
}

func TestLoadRejectsInvalidCrossFieldValues(t *testing.T) {
	setRequiredDirectories(t)
	t.Setenv("NAVIRE_EVENTS_ALLOWED_PROXIES", "127.0.0.1")
	t.Setenv("REDIS_RETRY_INTERVAL", "20s")
	t.Setenv("REDIS_MAX_WAIT", "10s")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "REDIS_RETRY_INTERVAL") {
		t.Fatalf("Load() error = %v, want retry interval error", err)
	}
}

func TestLoadRejectsInvalidProxy(t *testing.T) {
	setRequiredDirectories(t)
	t.Setenv("NAVIRE_EVENTS_ALLOWED_PROXIES", "not-a-proxy")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "NAVIRE_EVENTS_ALLOWED_PROXIES") {
		t.Fatalf("Load() error = %v, want proxy error", err)
	}
}

func TestLoadAcceptsMultipleAllowedProxies(t *testing.T) {
	setRequiredDirectories(t)
	t.Setenv("NAVIRE_EVENTS_ALLOWED_PROXIES", "127.0.0.1, 172.19.0.0/16")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.AllowedProxies) != 2 {
		t.Fatalf("AllowedProxies = %#v, want two entries", cfg.AllowedProxies)
	}
}

func TestLoadRejectsTrustedProxyWithoutAllowlist(t *testing.T) {
	setRequiredDirectories(t)
	t.Setenv("NAVIRE_EVENTS_ALLOWED_PROXIES", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "NAVIRE_EVENTS_ALLOWED_PROXIES") {
		t.Fatalf("Load() error = %v, want missing proxy allowlist error", err)
	}
}
