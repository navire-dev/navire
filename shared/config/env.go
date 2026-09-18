// Package config contains configuration primitives shared by Navire processes.
// Domain-specific structs and cross-field validation remain in each process.
package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Loader struct {
	errors []error
}

// NormalizeLogLevel returns the canonical name for a supported log level.
// warning is accepted as a compatibility alias for warn.
func NormalizeLogLevel(value string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "debug", "info", "warn", "error":
		return normalized, true
	case "warning":
		return "warn", true
	default:
		return "", false
	}
}

func ValidateLogLevel(key, value string) error {
	if _, ok := NormalizeLogLevel(value); ok {
		return nil
	}
	return fmt.Errorf("%s must be one of debug, info, warn, error", key)
}

func NewLoader() *Loader {
	return &Loader{}
}

func (l *Loader) String(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func (l *Loader) RequiredString(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		l.addError(fmt.Errorf("%s is required", key))
		return ""
	}
	return value
}

func (l *Loader) Int(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		l.addError(fmt.Errorf("%s=%q must be an integer: %w", key, value, err))
		return fallback
	}
	return parsed
}

func (l *Loader) Bool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		l.addError(fmt.Errorf("%s=%q must be a boolean: %w", key, value, err))
		return fallback
	}
	return parsed
}

func (l *Loader) Duration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := parseDuration(value)
	if err != nil {
		l.addError(fmt.Errorf("%s=%q must be a duration: %w", key, value, err))
		return fallback
	}
	return parsed
}

func parseDuration(value string) (time.Duration, error) {
	if strings.HasSuffix(value, "d") {
		days, err := strconv.ParseFloat(strings.TrimSuffix(value, "d"), 64)
		if err != nil || days <= 0 {
			if err == nil {
				err = fmt.Errorf("days must be greater than zero")
			}
			return 0, err
		}
		return time.Duration(days * float64(24*time.Hour)), nil
	}
	return time.ParseDuration(value)
}

func (l *Loader) AddError(err error) {
	if err != nil {
		l.addError(err)
	}
}

func (l *Loader) Err() error {
	return errors.Join(l.errors...)
}

func (l *Loader) addError(err error) {
	l.errors = append(l.errors, err)
}

// NormalizeListenAddr converts a port-only value into the address expected by
// net/http. Complete host:port values are preserved unchanged.
func NormalizeListenAddr(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if _, _, err := net.SplitHostPort(value); err == nil {
		return value
	}
	if _, err := strconv.Atoi(value); err == nil {
		return ":" + value
	}
	return value
}

func ValidateListenAddr(key, addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("%s=%q must be a valid host:port: %w", key, addr, err)
	}
	if host == "" && port == "" {
		return fmt.Errorf("%s must include a port", key)
	}
	parsed, err := strconv.Atoi(port)
	if err != nil || parsed < 0 || parsed > 65535 {
		if err == nil {
			err = fmt.Errorf("port out of range")
		}
		return fmt.Errorf("%s=%q has an invalid port: %w", key, addr, err)
	}
	return nil
}

// RedisAddr joins a Redis host and port using net.JoinHostPort.
func RedisAddr(host string, port int) string {
	return net.JoinHostPort(strings.TrimSpace(host), strconv.Itoa(port))
}

func ValidatePort(key string, port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s=%d must be between 1 and 65535", key, port)
	}
	return nil
}
