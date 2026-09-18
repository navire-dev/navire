package middleware

import (
	"context"
	"fmt"
	"os"

	"github.com/navire-dev/navire/navirectl/internal/config"
	"github.com/navire-dev/navire/navirectl/internal/runtime"
)

// RequireConfig loads the repository-local configuration before a command
// handler runs and makes it available through the explicit CLI runtime.
func RequireConfig(next Handler) Handler {
	return func(ctx context.Context, rt *runtime.Context) error {
		root, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve repository directory: %w", err)
		}
		cfg, err := config.Load(root)
		if err != nil {
			return err
		}
		rt.Values[config.RuntimeConfigKey] = cfg
		rt.Values[config.RuntimeRootKey] = root
		return next(ctx, rt)
	}
}
