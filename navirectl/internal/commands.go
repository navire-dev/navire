package internal

import (
	"github.com/urfave/cli/v3"

	"github.com/navire-dev/navire/navirectl/internal/middleware"
)

// CommandFactory is the stable registration seam for Navire CLI commands.
// Middleware will decorate factories here without coupling command discovery
// to command implementations.
type CommandFactory = middleware.CommandFactory

var defaultCommands = []CommandFactory{
	NewInitCmd,
	NewVersionCmd,
	middleware.UseMiddlewareChain(middleware.RequireConfig)(NewValidateCmd),
}

func RegisterSubCommands(root *cli.Command) {
	for _, factory := range defaultCommands {
		root.Commands = append(root.Commands, factory())
	}
}
