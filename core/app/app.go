package app

import (
	"fmt"

	"github.com/navire-dev/navire/core/internal/config"
	coreRuntime "github.com/navire-dev/navire/core/internal/runtime"
)

// New loads configuration and builds a Core runtime.
func New() (*coreRuntime.App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load configuration: %w", err)
	}
	return NewWithConfig(cfg)
}

// NewWithConfig builds a Core runtime from an explicit configuration.
func NewWithConfig(cfg *config.Config) (*coreRuntime.App, error) {
	return coreRuntime.New(cfg)
}
