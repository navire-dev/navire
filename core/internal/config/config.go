package config

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	sharedconfig "github.com/navire-dev/navire/shared/config"
	redisClient "github.com/navire-dev/navire/shared/redis"
)

const ( // mounted volume in the container
	InternalListenAddr = ":8080"
	RefreshInterval    = 24 * time.Hour

	// Timeouts HTTP
	DialTimeout           = 5 * time.Second
	TLSHandshakeTimeout   = 5 * time.Second
	ResponseHeaderTimeout = 30 * time.Second
	// Global Deadline by request (incl. download gzip)
	RequestDeadline = 5 * time.Minute

	// Download security limit
	MaxDownloadBytes = 40 * 1024 * 1024 // 40 MB

	// Event rate-limit internals. These are deliberately not environment knobs.
	EventRateLimitMaxEntries = 10000
	EventRateLimitSweep      = time.Minute
	EventRateLimitIdleTTL    = 15 * time.Minute
)

type EventRateLimitConfig struct {
	Burst             int
	RefillPerIPPerMin int
	MaxEntries        int
	SweepInterval     time.Duration
	IdleTTL           time.Duration
}

type Config struct {
	Version                  string        // ex: "0.0.1" or "dev"
	GitCommit                string        // ex: "3c137c7d7b4e3820a51ebf77c6152d6efc1b3f32"
	DataDir                  string        // ex: "/var/lib/navire"
	ShutdownTimeout          time.Duration // ex: 5s
	PlanRetention            time.Duration // ex: 30d
	PlanCleanupInterval      time.Duration // ex: 1h
	ReportingRetention       time.Duration // ex: 24h
	ReportingCleanupInterval time.Duration // ex: 1h
	LogLevel                 string        // "debug" | "info" | "warn" | "error"
	PrettyLog                bool          // true => zap dev (color), false => zap prod (JSON)
	AllowedCIDRS             []string      // optional, restrict access to specific IP (e.g. "1.2.3.4, 5.6.7.8")
	AllowedProxies           []string      // proxies allowed to provide forwarding headers
	TrustProxy               bool          // true => trust X-Forwarded-For headers (e.g. cloudflared)
	EventsRateLimit          *EventRateLimitConfig

	Redis redisClient.Config

	// Encryption
	SecretsEncryptionKey string // AES key for secrets stored in the runtime projection.
	DispatchPlanKey      string // AES key for URLs carried in Core-to-Dispatcher plans.
	RegistrationSecret   string // HMAC secret used to authenticate Dispatcher registration.
	TargetsDir           string // ex: "/config/targets"
	TemplatesDir         string // ex: "/config/templates"
}

func Load() (*Config, error) {
	loader := sharedconfig.NewLoader()
	allowedCIDRS, err := parseAllowedIPs(loader.String("NAVIRE_EVENTS_ALLOWED_CIDRS", "*"))
	if err != nil {
		loader.AddError(fmt.Errorf("NAVIRE_EVENTS_ALLOWED_CIDRS: %w", err))
	}
	allowedProxies, err := parseAllowedProxies(loader.String("NAVIRE_EVENTS_ALLOWED_PROXIES", ""))
	if err != nil {
		loader.AddError(fmt.Errorf("NAVIRE_EVENTS_ALLOWED_PROXIES: %w", err))
	}
	eventsRateLimit := loadEventRateLimit(loader)
	cfg := &Config{
		Version:                  loader.String("NAVIRE_VERSION", "dev"),
		GitCommit:                loader.String("GIT_COMMIT", "unknown"),
		DataDir:                  loader.String("NAVIRE_DATA_DIR", "/var/lib/navire"),
		ShutdownTimeout:          loader.Duration("NAVIRE_SHUTDOWN_TIMEOUT", 5*time.Second),
		PlanRetention:            loader.Duration("NAVIRE_PLAN_RETENTION", 30*24*time.Hour),
		PlanCleanupInterval:      loader.Duration("NAVIRE_PLAN_CLEANUP_INTERVAL", time.Hour),
		ReportingRetention:       loader.Duration("NAVIRE_REPORTING_RETENTION", 24*time.Hour),
		ReportingCleanupInterval: loader.Duration("NAVIRE_REPORTING_CLEANUP_INTERVAL", time.Hour),
		LogLevel:                 loader.String("NAVIRE_LOG_LEVEL", "info"),
		PrettyLog:                loader.Bool("NAVIRE_PRETTY_LOG", true),
		AllowedCIDRS:             allowedCIDRS,
		AllowedProxies:           allowedProxies,
		TrustProxy:               loader.Bool("NAVIRE_EVENTS_TRUST_PROXY", true),
		EventsRateLimit:          eventsRateLimit,
		Redis:                    redisClient.Load(loader),
		SecretsEncryptionKey:     loader.RequiredString("NAVIRE_SECRETS_ENCRYPTION_KEY"),
		DispatchPlanKey:          loader.RequiredString("NAVIRE_DISPATCH_PLAN_KEY"),
		RegistrationSecret:       loader.RequiredString("NAVIRE_DSPC_REGISTRATION_SECRET"),
		TargetsDir:               loader.String("NAVIRE_TARGETS_DIR", "/config/targets"),
		TemplatesDir:             loader.String("NAVIRE_TEMPLATES_DIR", "/config/templates"),
	}
	if err := loader.Err(); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadEventRateLimit(loader *sharedconfig.Loader) *EventRateLimitConfig {
	burstValue, burstSet := os.LookupEnv("NAVIRE_EVENTS_RATE_LIMIT_BURST")
	refillValue, refillSet := os.LookupEnv("NAVIRE_EVENTS_RATE_LIMIT_REFILL_PER_MIN")
	burstSet = burstSet && strings.TrimSpace(burstValue) != ""
	refillSet = refillSet && strings.TrimSpace(refillValue) != ""

	if !burstSet && !refillSet {
		return nil
	}
	if burstSet != refillSet {
		loader.AddError(fmt.Errorf("NAVIRE_EVENTS_RATE_LIMIT_BURST and NAVIRE_EVENTS_RATE_LIMIT_REFILL_PER_MIN must be configured together"))
		return nil
	}

	burst := loader.Int("NAVIRE_EVENTS_RATE_LIMIT_BURST", 0)
	refill := loader.Int("NAVIRE_EVENTS_RATE_LIMIT_REFILL_PER_MIN", 0)
	if burst <= 0 {
		loader.AddError(fmt.Errorf("NAVIRE_EVENTS_RATE_LIMIT_BURST must be > 0 when rate limiting is enabled"))
	}
	if refill <= 0 {
		loader.AddError(fmt.Errorf("NAVIRE_EVENTS_RATE_LIMIT_REFILL_PER_MIN must be > 0 when rate limiting is enabled"))
	}

	return &EventRateLimitConfig{
		Burst:             burst,
		RefillPerIPPerMin: refill,
		MaxEntries:        EventRateLimitMaxEntries,
		SweepInterval:     EventRateLimitSweep,
		IdleTTL:           EventRateLimitIdleTTL,
	}
}

func (c *Config) Validate() error {
	if err := c.validateCore(); err != nil {
		return err
	}
	if err := c.validateRedis(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateCore() error {
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("NAVIRE_SHUTDOWN_TIMEOUT must be > 0")
	}
	if c.PlanRetention <= 0 {
		return fmt.Errorf("NAVIRE_PLAN_RETENTION must be > 0")
	}
	if c.PlanCleanupInterval <= 0 {
		return fmt.Errorf("NAVIRE_PLAN_CLEANUP_INTERVAL must be > 0")
	}
	if c.ReportingRetention <= 0 {
		return fmt.Errorf("NAVIRE_REPORTING_RETENTION must be > 0")
	}
	if c.ReportingCleanupInterval <= 0 {
		return fmt.Errorf("NAVIRE_REPORTING_CLEANUP_INTERVAL must be > 0")
	}
	if c.TrustProxy && len(c.AllowedProxies) == 0 {
		return fmt.Errorf("NAVIRE_EVENTS_ALLOWED_PROXIES is required when NAVIRE_EVENTS_TRUST_PROXY is true")
	}
	return sharedconfig.ValidateLogLevel("NAVIRE_LOG_LEVEL", c.LogLevel)
}

func (c *Config) validateRedis() error {
	return c.Redis.Validate()
}

func parseAllowedIPs(allowed string) ([]string, error) {
	parts := splitAndTrim(allowed, ",")
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "*") {
		return []string{"*"}, nil
	}
	for _, part := range parts {
		if part == "*" {
			return nil, fmt.Errorf("'*' cannot be combined with IPs or CIDRs")
		}
		if _, _, err := net.ParseCIDR(part); err != nil {
			if net.ParseIP(part) == nil {
				return nil, fmt.Errorf("invalid IP or CIDR %q", part)
			}
		}
	}
	return parts, nil
}

func parseAllowedProxies(allowed string) ([]string, error) {
	parts := splitAndTrim(allowed, ",")
	for _, part := range parts {
		if _, _, err := net.ParseCIDR(part); err == nil {
			continue
		}
		if net.ParseIP(part) == nil {
			return nil, fmt.Errorf("invalid proxy IP or CIDR %q", part)
		}
	}
	return parts, nil
}

func splitAndTrim(s, sep string) []string {
	if s == "" {
		return nil
	}
	raw := strings.Split(s, sep)
	parts := make([]string, 0, len(raw)) // preallocate to number of parts
	for _, part := range raw {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
