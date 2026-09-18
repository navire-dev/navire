package navirectl

import (
	"context"

	internal "github.com/navire-dev/navire/navirectl/internal"
)

// Execute runs the Navire CLI.
func Execute(ctx context.Context, args []string) error {
	return internal.Execute(ctx, args)
}
