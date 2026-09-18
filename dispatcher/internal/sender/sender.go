// Package sender renders execution plans and sends notifications to providers.
package sender

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	messageRender "github.com/navire-dev/navire/dispatcher/internal/render"
	"github.com/navire-dev/navire/dispatcher/internal/sinks"
	"github.com/navire-dev/navire/dispatcher/internal/sinks/contract"
	sharedcrypto "github.com/navire-dev/navire/shared/crypto"
	"github.com/navire-dev/navire/shared/logger"
	"github.com/navire-dev/navire/shared/models"
	sharedproviders "github.com/navire-dev/navire/shared/providers"
)

type Service struct {
	HTTPClient      *http.Client
	DispatchPlanKey []byte
	Registry        *sinks.Registry
	Logger          logger.Logger
}

type Result struct {
	TargetName string
	Provider   sharedproviders.ID
	Duration   time.Duration
	Err        error
}

func (r Result) Error() error {
	return r.Err
}

func (s Service) Execute(ctx context.Context, plan models.ExecutionPlan) (result Result) {
	started := time.Now()
	target := plan.Target
	result = Result{TargetName: target.Name, Provider: target.Provider}
	defer func() { result.Duration = time.Since(started) }()

	decryptedURL, err := sharedcrypto.Decrypt(target.URL, s.DispatchPlanKey)
	if err != nil {
		result.Err = fmt.Errorf("decrypt target %s: %w", target.Name, err)
		return result
	}

	rendered, err := messageRender.Render(plan.Variant.Title, plan.Variant.Body, plan.Data)
	if err != nil {
		result.Err = fmt.Errorf("render target %s: %w", target.Name, err)
		return result
	}
	rendered.Priority = plan.Variant.Priority

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
	for attempt := 1; attempt <= policy.Retry.MaxAttempts; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		req, err := http.NewRequestWithContext(attemptCtx, prepared.Method, prepared.URL, bytes.NewReader(prepared.Body))
		if err == nil {
			req.Header = prepared.Headers.Clone()
			var resp *http.Response
			resp, err = s.HTTPClient.Do(req)
			if err == nil {
				if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
					_ = resp.Body.Close()
					err = fmt.Errorf("%s returned %s", prepared.Provider, resp.Status)
				} else {
					_ = resp.Body.Close()
				}
			} else {
				err = sanitizeHTTPError(err)
			}
		} else {
			err = fmt.Errorf("create %s request", prepared.Provider)
		}
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt == policy.Retry.MaxAttempts || ctx.Err() != nil {
			break
		}
		delay := retryDelay(wait, policy.Retry.Backoff, attempt)
		if s.Logger != nil {
			s.Logger.Warn("delivery attempt failed, retrying",
				logger.String("target", safeURLForLog(prepared.URL)),
				logger.String("provider", prepared.Provider),
				logger.Int("attempt", attempt),
				logger.Int("max_attempts", policy.Retry.MaxAttempts),
				logger.Duration("retry_in", delay),
				logger.Error(err))
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
	return fmt.Errorf("failed after %d attempts: %w", policy.Retry.MaxAttempts, lastErr)
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
