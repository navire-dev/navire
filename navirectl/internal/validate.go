package internal

import (
	"github.com/navire-dev/navire/navirectl/internal/command"
	validatehandler "github.com/navire-dev/navire/navirectl/internal/validate"
	"github.com/urfave/cli/v3"
)

func NewValidateCmd() *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate Navire configuration",
		Commands: []*cli.Command{
			command.New("templates", "Validate template definitions", validatehandler.Templates),
		},
	}
}
