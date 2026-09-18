package version

import (
	"context"

	"github.com/navire-dev/navire/navirectl/internal/runtime"
	sharedversion "github.com/navire-dev/navire/shared/version"
)

// New renders the detailed human-readable version output.
func New(_ context.Context, rt *runtime.Context) error {
	log := rt.Logger
	log.Info("ℹ navirectl %s", sharedversion.Version)
	log.Info("ℹ Shipping events without a shipping department.")
	if rt.Bool("debug") {
		log.Info("ℹ The captain will always go down with the Core.")
	}
	log.Info("ℹ Commit: %s", sharedversion.Commit)
	log.Info("ℹ Built: %s", sharedversion.BuildDate)
	log.Info("ℹ Go: %s", sharedversion.GoVersion)
	return nil
}
