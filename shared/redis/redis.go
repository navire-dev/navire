package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	sharedconfig "github.com/navire-dev/navire/shared/config"
	"github.com/navire-dev/navire/shared/logger"
	"github.com/redis/go-redis/v9"
)

// Config contains the Redis connection and retry settings shared by Core and DSPC.
type Config struct {
	URI                   string
	Addr                  string
	User                  string
	Password              string
	DB                    int
	TLSEnabled            bool
	TLSInsecureSkipVerify bool
	DialTimeout           time.Duration
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	PoolSize              int
	RetryInterval         time.Duration
	MaxWait               time.Duration
	PingTimeout           time.Duration
	WarnThreshold         int
}

// Load reads the common Redis environment variables into cfg.
func Load(cfg *sharedconfig.Loader) Config {
	uri := strings.TrimSpace(cfg.String("REDIS_URI", ""))
	result := Config{
		URI:                   uri,
		TLSEnabled:            cfg.Bool("REDIS_TLS_ENABLED", false),
		TLSInsecureSkipVerify: cfg.Bool("REDIS_TLS_INSECURE_SKIP_VERIFY", false),
		DialTimeout:           cfg.Duration("REDIS_DIAL_TIMEOUT", 5*time.Second),
		ReadTimeout:           cfg.Duration("REDIS_READ_TIMEOUT", 3*time.Second),
		WriteTimeout:          cfg.Duration("REDIS_WRITE_TIMEOUT", 3*time.Second),
		PoolSize:              cfg.Int("REDIS_POOL_SIZE", 10),
		RetryInterval:         cfg.Duration("REDIS_RETRY_INTERVAL", 2*time.Second),
		MaxWait:               cfg.Duration("REDIS_MAX_WAIT", 10*time.Second),
		PingTimeout:           cfg.Duration("REDIS_PING_TIMEOUT", 5*time.Second),
		WarnThreshold:         cfg.Int("REDIS_WARN_THRESHOLD", 3),
	}

	if uri != "" {
		if conflicts := configuredSplitVariables(); len(conflicts) > 0 {
			cfg.AddError(fmt.Errorf("REDIS_URI cannot be combined with %s", strings.Join(conflicts, ", ")))
		}
		parsed, err := redis.ParseURL(uri)
		if err != nil {
			cfg.AddError(fmt.Errorf("REDIS_URI=%q must be a valid Redis URI: %w", uri, err))
			return result
		}
		result.Addr = parsed.Addr
		result.User = parsed.Username
		result.Password = parsed.Password
		result.DB = parsed.DB
		result.TLSEnabled = parsed.TLSConfig != nil
		return result
	}

	result.Addr = sharedconfig.RedisAddr(cfg.RequiredString("REDIS_ADDR"), cfg.Int("REDIS_PORT", 6379))
	result.User = cfg.String("REDIS_USER", "")
	result.Password = cfg.String("REDIS_PASSWORD", "")
	result.DB = cfg.Int("REDIS_DB", 0)
	return result
}

func configuredSplitVariables() []string {
	keys := []string{
		"REDIS_ADDR",
		"REDIS_PORT",
		"REDIS_USER",
		"REDIS_PASSWORD",
		"REDIS_DB",
		"REDIS_TLS_ENABLED",
		"REDIS_TLS_INSECURE_SKIP_VERIFY",
	}
	configured := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, ok := os.LookupEnv(key); ok {
			configured = append(configured, key)
		}
	}
	return configured
}

// Validate checks the common Redis settings and address.
func (c Config) Validate() error {
	if strings.TrimSpace(c.URI) != "" {
		if c.TLSInsecureSkipVerify {
			return fmt.Errorf("REDIS_TLS_INSECURE_SKIP_VERIFY cannot be used with REDIS_URI")
		}
		parsed, err := redis.ParseURL(c.URI)
		if err != nil {
			return fmt.Errorf("REDIS_URI=%q must be a valid Redis URI: %w", c.URI, err)
		}
		if parsed.Addr == "" {
			return fmt.Errorf("REDIS_URI must include a Redis host")
		}
	} else {
		if err := c.validateAddress(); err != nil {
			return err
		}
	}
	return c.validateCommon()
}

func (c Config) validateAddress() error {
	host, port, err := net.SplitHostPort(c.Addr)
	if err != nil || host == "" {
		return fmt.Errorf("REDIS_ADDR and REDIS_PORT must define a valid Redis address: %q", c.Addr)
	}
	parsedPort, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("REDIS_PORT=%q must be an integer", port)
	}
	if err := sharedconfig.ValidatePort("REDIS_PORT", parsedPort); err != nil {
		return err
	}
	return nil
}

func (c Config) validateCommon() error {
	if c.DB < 0 {
		return fmt.Errorf("REDIS_DB must be >= 0")
	}
	if c.PoolSize <= 0 {
		return fmt.Errorf("REDIS_POOL_SIZE must be > 0")
	}
	if c.DialTimeout <= 0 || c.ReadTimeout <= 0 || c.WriteTimeout <= 0 || c.MaxWait <= 0 || c.PingTimeout <= 0 || c.RetryInterval <= 0 {
		return fmt.Errorf("redis timeout and retry values must be > 0")
	}
	if c.WarnThreshold < 0 {
		return fmt.Errorf("REDIS_WARN_THRESHOLD must be >= 0")
	}
	if !c.TLSEnabled && c.TLSInsecureSkipVerify {
		return fmt.Errorf("REDIS_TLS_INSECURE_SKIP_VERIFY requires REDIS_TLS_ENABLED=true")
	}
	if c.RetryInterval > c.MaxWait {
		return fmt.Errorf("REDIS_RETRY_INTERVAL must be <= REDIS_MAX_WAIT")
	}
	return nil
}

// retryConfig holds retry policy settings.
type retryConfig struct {
	maxWait       time.Duration
	pingTimeout   time.Duration
	initialWait   time.Duration
	warnThreshold int // warn after this many attempts
}

// connectionLogger handles all Redis connection logging.
type connectionLogger struct {
	logger logger.Logger
}

func (cl *connectionLogger) logConnectionStart(addr string, retryInterval, maxWait time.Duration) {
	cl.logger.Info("connecting to redis",
		logger.String("addr", addr),
		logger.Duration("retry_interval", retryInterval),
		logger.Duration("max_retry_wait", maxWait))
}

func (cl *connectionLogger) logSuccess(addr string, attempts int, elapsed time.Duration) {
	if attempts > 1 {
		cl.logger.Warn("connected to redis after retry",
			logger.String("addr", addr),
			logger.Int("attempts", attempts),
			logger.Duration("elapsed", elapsed))
	} else {
		cl.logger.Info("connected to redis",
			logger.String("addr", addr))
	}
}

func (cl *connectionLogger) logRetry(addr string, attempt int, nextRetry time.Duration, warnThreshold int, err error) {
	switch {
	case attempt <= warnThreshold:
		cl.logger.Warn("redis connection failed, retrying",
			logger.String("addr", addr),
			logger.Int("attempt", attempt),
			logger.Duration("next_retry_in", nextRetry),
			logger.Error(err))
	default:
		cl.logger.Error("redis still unavailable - connection attempts failing",
			logger.String("addr", addr),
			logger.Int("attempt", attempt),
			logger.Duration("next_retry_in", nextRetry),
			logger.Error(err))
	}
}

// NewClient creates a configured Redis client without performing network I/O.
func NewClient(opts Config) (*redis.Client, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	redisOptions := &redis.Options{
		Addr:         opts.Addr,
		Username:     opts.User,
		Password:     opts.Password,
		DB:           opts.DB,
		DialTimeout:  opts.DialTimeout,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		PoolSize:     opts.PoolSize,
	}
	if opts.URI != "" {
		parsed, err := redis.ParseURL(opts.URI)
		if err != nil {
			return nil, fmt.Errorf("parse REDIS_URI: %w", err)
		}
		redisOptions = parsed
		redisOptions.DialTimeout = opts.DialTimeout
		redisOptions.ReadTimeout = opts.ReadTimeout
		redisOptions.WriteTimeout = opts.WriteTimeout
		redisOptions.PoolSize = opts.PoolSize
	}
	if opts.URI == "" && opts.TLSEnabled {
		redisOptions.TLSConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: opts.TLSInsecureSkipVerify, //nolint:gosec // explicitly configured by the operator for controlled environments
		}
	}
	return redis.NewClient(redisOptions), nil
}

// WaitForReady retries Redis connectivity indefinitely until it succeeds or ctx is cancelled.
func WaitForReady(ctx context.Context, client *redis.Client, opts Config, log logger.Logger) error {
	if client == nil {
		return fmt.Errorf("redis client is required")
	}
	if err := opts.Validate(); err != nil {
		return err
	}
	retry := retryConfig{
		maxWait:       opts.MaxWait,
		pingTimeout:   opts.PingTimeout,
		initialWait:   opts.RetryInterval,
		warnThreshold: opts.WarnThreshold,
	}
	return connectWithRetry(ctx, client, opts.Addr, retry, &connectionLogger{logger: log})
}

// connectWithRetry handles the retry loop with exponential backoff.
func connectWithRetry(ctx context.Context, client *redis.Client, addr string, retry retryConfig, log *connectionLogger) error {
	started := time.Now()
	log.logConnectionStart(addr, retry.initialWait, retry.maxWait)
	attempt := 0
	wait := retry.initialWait

	for {
		attempt++

		// Attempt connection
		pingCtx, pingCancel := context.WithTimeout(ctx, retry.pingTimeout)
		err := client.Ping(pingCtx).Err()
		pingCancel()

		if err == nil {
			log.logSuccess(addr, attempt, time.Since(started))
			return nil
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return fmt.Errorf("wait for redis at %s: %w", addr, ctx.Err())
		case <-timer.C:
			log.logRetry(addr, attempt, wait, retry.warnThreshold, err)
			wait *= 2
			if wait > retry.maxWait {
				wait = retry.maxWait
			}
		}
	}
}
