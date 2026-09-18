package ingest

import (
	"fmt"

	"github.com/navire-dev/navire/shared/models"
	"github.com/navire-dev/navire/shared/providers"
)

// TargetsFile represents a targets/*.yml file (e.g., targets/gotify.yml)
type TargetsFile struct {
	Provider  providers.ID `yaml:"provider"`  // Provider slug from the shared catalog.
	Enabled   bool         `yaml:"enabled"`   // Global enable/disable for all endpoints
	Endpoints []Endpoint   `yaml:"endpoints"` // List of endpoints (multi-instance support)
}

type (
	Endpoint = models.Endpoint
	Auth     = models.Auth
)

// Validate checks if the TargetsFile is valid.
func (tf *TargetsFile) Validate() error {
	if tf.Provider == "" {
		return fmt.Errorf("provider type is required")
	}

	if err := providers.Validate(string(tf.Provider)); err != nil {
		return err
	}

	if len(tf.Endpoints) == 0 {
		return fmt.Errorf("at least one endpoint is required")
	}

	// Validate each endpoint
	names := make(map[string]bool)
	for i, ep := range tf.Endpoints {
		if err := ep.Validate(); err != nil {
			return fmt.Errorf("endpoints[%d] (%s): %w", i, ep.Key, err)
		}

		// Check for duplicate keys within the same file
		if names[ep.Key] {
			return fmt.Errorf("duplicate endpoint key %q in same file", ep.Key)
		}
		names[ep.Key] = true
	}

	return nil
}
