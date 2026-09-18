package internal

import (
	"github.com/urfave/cli/v3"

	"github.com/navire-dev/navire/navirectl/internal/command"
	inithand "github.com/navire-dev/navire/navirectl/internal/init"
)

// NewInitCmd creates the repository initialization command.
func NewInitCmd() *cli.Command {
	return command.New("init", "Initialize a Navire configuration repository", inithand.New)
}
