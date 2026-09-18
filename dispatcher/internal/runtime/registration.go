package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/navire-dev/navire/dispatcher/internal/config"
	"github.com/navire-dev/navire/shared/dispatchauth"
	"github.com/navire-dev/navire/shared/logger"
	sharedruntime "github.com/navire-dev/navire/shared/runtime"
)

func (a *App) registerUntilAvailable(ctx context.Context) {
	for {
		if !a.registerOnceUntilAvailable(ctx) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-a.registrationWake:
		}
	}
}

func (a *App) registerOnceUntilAvailable(ctx context.Context) bool {
	delay := registrationRetryInitial
	for {
		token, err := register(ctx, a.cfg, a.registrationSecret, a.registrationClient)
		if err == nil {
			a.registrationMu.Lock()
			a.registrationToken = token
			a.registrationMu.Unlock()
			a.logger.Info("dispatcher registered with Core", logger.String("consumer", a.cfg.ConsumerID))
			return true
		}
		if ctx.Err() != nil {
			return false
		}
		a.logger.Warn("dispatcher registration unavailable, retrying",
			logger.String("consumer", a.cfg.ConsumerID),
			logger.Duration("retry_in", delay),
			logger.Error(err))
		if !a.waitForRegistrationRetry(ctx, delay) {
			return false
		}
		delay = sharedruntime.NextRetryDelay(delay, registrationRetryMax)
	}
}

func (a *App) waitForRegistrationRetry(ctx context.Context, delay time.Duration) bool {
	if a.registrationWake == nil {
		return sharedruntime.WaitForRetry(ctx, delay)
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	case <-a.registrationWake:
		return true
	}
}

func (a *App) invalidateRegistration() {
	a.registrationMu.Lock()
	a.registrationToken = ""
	a.registrationMu.Unlock()
	a.requestRegistration()
}

func (a *App) requestRegistration() {
	if a.registrationWake == nil {
		return
	}
	select {
	case a.registrationWake <- struct{}{}:
	default:
	}
}

func (a *App) currentRegistrationToken() string {
	a.registrationMu.RLock()
	defer a.registrationMu.RUnlock()
	return a.registrationToken
}

func register(ctx context.Context, cfg *config.Config, secret []byte, client *http.Client) (string, error) {
	nonce, err := dispatchauth.NewNonce()
	if err != nil {
		return "", err
	}
	timestamp := time.Now().Unix()
	request := dispatchauth.RegistrationRequest{
		DispatcherID: cfg.ConsumerID,
		Timestamp:    timestamp,
		Nonce:        nonce,
		Signature:    dispatchauth.Sign(secret, cfg.ConsumerID, timestamp, nonce),
	}
	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("encode registration request: %w", err)
	}
	endpoint := strings.TrimRight(cfg.CoreURL, "/") + "/api/register-dispatcher"
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create registration request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		return "", fmt.Errorf("send registration request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("core rejected registration with HTTP %s", response.Status)
	}
	var result dispatchauth.RegistrationResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode registration response: %w", err)
	}
	if result.Token == "" {
		return "", fmt.Errorf("core returned an empty registration token")
	}
	return result.Token, nil
}
