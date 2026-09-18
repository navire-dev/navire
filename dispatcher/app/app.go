package app

import (
	"fmt"

	"github.com/navire-dev/navire/dispatcher/internal/config"
	dispatcherRuntime "github.com/navire-dev/navire/dispatcher/internal/runtime"
)

// New loads configuration and builds a Dispatcher runtime.
func New() (*dispatcherRuntime.App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load dispatcher configuration: %w", err)
	}
	return NewWithConfig(cfg)
}

// NewWithConfig builds a Dispatcher runtime from an explicit configuration.
func NewWithConfig(cfg *config.Config) (*dispatcherRuntime.App, error) {
	return dispatcherRuntime.New(cfg)
}
