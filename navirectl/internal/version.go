package internal

import (
	"github.com/urfave/cli/v3"

	"github.com/navire-dev/navire/navirectl/internal/command"
	versionhandler "github.com/navire-dev/navire/navirectl/internal/version"
)

// NewVersionCmd creates the version command and wires its handler.
func NewVersionCmd() *cli.Command {
	cmd := command.New("version", "Print Navire CLI version information", versionhandler.New, command.BindBool("debug"))
	cmd.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Print the debug easter egg",
		},
	}
	return cmd
}
