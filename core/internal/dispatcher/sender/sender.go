// Package sender renders execution plans and sends notifications to providers.
package sender

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	messageRender "github.com/navire-dev/navire/core/internal/dispatcher/render"
	"github.com/navire-dev/navire/core/internal/dispatcher/sinks"
	"github.com/navire-dev/navire/core/internal/dispatcher/sinks/contract"
	sharedcrypto "github.com/navire-dev/navire/shared/crypto"
	"github.com/navire-dev/navire/shared/logger"
	"github.com/navire-dev/navire/shared/models"
	sharedproviders "github.com/navire-dev/navire/shared/providers"
)

type Service struct {
	HTTPClient       *http.Client
	ExecutionPlanKey []byte
	Registry         *sinks.Registry
	Logger           logger.Logger
}

type Result struct {
	TargetName string
	Provider   sharedproviders.ID
	Duration   time.Duration
	Err        error
}

// DeliveryError preserves provider response metadata across the dispatcher
// boundary so callers can log and report delivery failures precisely.
type DeliveryError struct {
	Provider      string
	StatusCode    int
	Status        string
	RetryAfter    time.Duration
	HasRetryAfter bool
	Attempts      int
	MaxAttempts   int
	Err           error
}

type attemptResult struct {
	err           error
	statusCode    int
	status        string
	retryAfter    time.Duration
	hasRetryAfter bool
}

func (e *DeliveryError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.Err.Error()
}

func (e *DeliveryError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (r Result) Error() error {
	return r.Err
}

func (s Service) Execute(ctx context.Context, plan models.ExecutionPlan) (result Result) {
	started := time.Now()
	target := plan.Target
	result = Result{TargetName: target.Name, Provider: target.Provider}
	defer func() { result.Duration = time.Since(started) }()

	decryptedURL, err := sharedcrypto.Decrypt(target.URL, s.ExecutionPlanKey)
	if err != nil {
		result.Err = fmt.Errorf("decrypt target %s: %w", target.Name, err)
		return result
	}
	if s.Logger != nil {
		s.Logger.Info("dispatcher target decrypted",
			logger.String("target", target.Name),
			logger.String("provider", string(target.Provider)),
			logger.String("url", safeURLForLog(decryptedURL)))
	}

	rendered, err := messageRender.Render(plan.Variant.Title, plan.Variant.Body, plan.Data)
	if err != nil {
		result.Err = fmt.Errorf("render target %s: %w", target.Name, err)
		return result
	}
	rendered.Priority = plan.Variant.Priority
	if s.Logger != nil {
		s.Logger.Info("dispatcher message rendered",
			logger.String("target", target.Name),
			logger.String("provider", string(target.Provider)))
	}

	provider, err := s.Registry.Resolve(target.Provider)
	if err != nil {
		result.Err = err
		return result
	}
	endpoint := models.Endpoint{
		Provider: target.Provider,
		FullName: target.Name,
		URL:      decryptedURL,
		Auth:     models.Auth{Type: models.AuthNone},
	}
	prepared, err := provider.Prepare(endpoint, rendered)
	if err != nil {
		result.Err = fmt.Errorf("prepare target %s: %w", target.Name, err)
		return result
	}
	if s.Logger != nil {
		s.Logger.Info("dispatcher request prepared",
			logger.String("target", target.Name),
			logger.String("provider", string(target.Provider)),
			logger.String("method", prepared.Method),
			logger.String("url", safeURLForLog(prepared.URL)))
	}
	if err := s.sendWithPolicy(ctx, prepared, target.Policy); err != nil {
		result.Err = fmt.Errorf("deliver to %s: %w", target.Name, err)
	}
	return result
}

func (s Service) sendWithPolicy(ctx context.Context, prepared contract.PreparedRequest, policy models.DeliveryPolicy) error {
	policy, err := normalizePolicy(policy)
	if err != nil {
		return err
	}
	timeout, _ := time.ParseDuration(policy.Timeout)
	wait, _ := time.ParseDuration(policy.Retry.InitialWait)
	var lastErr error
	var lastStatusCode int
	var lastStatus string
	var lastRetryAfter time.Duration
	var hasLastRetryAfter bool
	for attempt := 1; attempt <= policy.Retry.MaxAttempts; attempt++ {
		if s.Logger != nil {
			s.logAttemptStarted(prepared, attempt, policy.Retry.MaxAttempts, timeout)
		}
		result := s.sendAttempt(ctx, prepared, timeout)
		if result.err == nil {
			if s.Logger != nil {
				s.logAttemptSucceeded(prepared, attempt)
			}
			return nil
		}
		if s.Logger != nil {
			s.logAttemptFailure(prepared, attempt, policy.Retry.MaxAttempts, result)
		}
		lastErr = result.err
		lastStatusCode = result.statusCode
		lastStatus = result.status
		lastRetryAfter = result.retryAfter
		hasLastRetryAfter = result.hasRetryAfter
		if attempt == policy.Retry.MaxAttempts || ctx.Err() != nil {
			break
		}
		delay := retryDelay(wait, policy.Retry.Backoff, attempt)
		if result.hasRetryAfter {
			delay = result.retryAfter
		}
		if s.Logger != nil {
			s.logAttemptRetry(prepared, attempt, policy.Retry.MaxAttempts, result, delay)
		}
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return &DeliveryError{
		Provider:      prepared.Provider,
		StatusCode:    lastStatusCode,
		Status:        lastStatus,
		RetryAfter:    lastRetryAfter,
		HasRetryAfter: hasLastRetryAfter,
		Attempts:      policy.Retry.MaxAttempts,
		MaxAttempts:   policy.Retry.MaxAttempts,
		Err:           fmt.Errorf("failed after %d attempts: %w", policy.Retry.MaxAttempts, lastErr),
	}
}

func (s Service) sendAttempt(ctx context.Context, prepared contract.PreparedRequest, timeout time.Duration) attemptResult {
	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(attemptCtx, prepared.Method, prepared.URL, bytes.NewReader(prepared.Body))
	if err != nil {
		return attemptResult{err: fmt.Errorf("create %s request", prepared.Provider)}
	}
	req.Header = prepared.Headers.Clone()
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return attemptResult{err: sanitizeHTTPError(err)}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return attemptResult{}
	}
	serverRetryAfter, hasServerRetryAfter := retryAfter(resp.Header.Get("Retry-After"), time.Now())
	return attemptResult{
		err:           fmt.Errorf("%s returned %s", prepared.Provider, resp.Status),
		statusCode:    resp.StatusCode,
		status:        resp.Status,
		retryAfter:    serverRetryAfter,
		hasRetryAfter: hasServerRetryAfter,
	}
}

func (s Service) logAttemptStarted(prepared contract.PreparedRequest, attempt, maxAttempts int, timeout time.Duration) {
	s.Logger.Info("dispatcher delivery attempt started",
		logger.String("target", safeURLForLog(prepared.URL)),
		logger.String("provider", prepared.Provider),
		logger.Int("attempt", attempt),
		logger.Int("max_attempts", maxAttempts),
		logger.Duration("timeout", timeout))
}

func (s Service) logAttemptSucceeded(prepared contract.PreparedRequest, attempt int) {
	s.Logger.Info("dispatcher delivery attempt succeeded",
		logger.String("target", safeURLForLog(prepared.URL)),
		logger.String("provider", prepared.Provider),
		logger.Int("attempt", attempt))
}

func (s Service) logAttemptFailure(prepared contract.PreparedRequest, attempt, maxAttempts int, result attemptResult) {
	fields := attemptLogFields(prepared, attempt, maxAttempts)
	fields = append(fields, logger.Error(result.err))
	appendHTTPFailureFields(&fields, result.statusCode, result.status, result.retryAfter, result.hasRetryAfter)
	s.Logger.Warn("dispatcher delivery attempt failed", fields...)
}

func (s Service) logAttemptRetry(prepared contract.PreparedRequest, attempt, maxAttempts int, result attemptResult, delay time.Duration) {
	fields := attemptLogFields(prepared, attempt, maxAttempts)
	fields = append(fields, logger.Duration("retry_in", delay), logger.Error(result.err))
	appendHTTPFailureFields(&fields, result.statusCode, result.status, result.retryAfter, result.hasRetryAfter)
	s.Logger.Warn("delivery attempt failed, retrying", fields...)
}

func attemptLogFields(prepared contract.PreparedRequest, attempt, maxAttempts int) []zap.Field {
	return []zap.Field{
		logger.String("target", safeURLForLog(prepared.URL)),
		logger.String("provider", prepared.Provider),
		logger.Int("attempt", attempt),
		logger.Int("max_attempts", maxAttempts),
	}
}

func safeURLForLog(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "<invalid endpoint>"
	}
	return parsed.Scheme + "://" + parsed.Host
}

func sanitizeHTTPError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("%s request failed: %w", urlErr.Op, urlErr.Err)
	}
	return err
}

func normalizePolicy(policy models.DeliveryPolicy) (models.DeliveryPolicy, error) {
	if policy.Timeout == "" {
		policy.Timeout = "10s"
	}
	if policy.Retry.MaxAttempts == 0 {
		policy.Retry.MaxAttempts = 1
	}
	if policy.Retry.Backoff == "" {
		policy.Retry.Backoff = models.BackoffNone
	}
	return policy, policy.Validate()
}

func retryDelay(initial time.Duration, strategy models.BackoffStrategy, attempt int) time.Duration {
	const maxDelay = 5 * time.Minute
	switch strategy {
	case models.BackoffLinear:
		delay := initial * time.Duration(attempt)
		if delay > maxDelay {
			return maxDelay
		}
		return delay
	case models.BackoffExponential:
		delay := initial
		for step := 1; step < attempt; step++ {
			if delay > maxDelay/2 {
				return maxDelay
			}
			delay *= 2
		}
		if delay > maxDelay {
			return maxDelay
		}
		return delay
	default:
		return 0
	}
}

func retryAfter(value string, now time.Time) (time.Duration, bool) {
	const maxDelay = 5 * time.Minute

	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
		if seconds > int64(maxDelay/time.Second) {
			return maxDelay, true
		}
		return cappedRetryDelay(time.Duration(seconds) * time.Second), true
	}
	when, err := http.ParseTime(value)
	if err != nil || !when.After(now) {
		return 0, false
	}
	return cappedRetryDelay(when.Sub(now)), true
}

func cappedRetryDelay(delay time.Duration) time.Duration {
	const maxDelay = 5 * time.Minute
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

func appendHTTPFailureFields(fields *[]zap.Field, statusCode int, status string, retryAfter time.Duration, hasRetryAfter bool) {
	if statusCode != 0 {
		*fields = append(*fields,
			logger.Int("status_code", statusCode),
			logger.String("status", status),
		)
	}
	if hasRetryAfter {
		*fields = append(*fields, logger.Duration("retry_after", retryAfter))
	}
}
