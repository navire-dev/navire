package internal

import (
	"context"
	"fmt"

	sharedconfig "github.com/navire-dev/navire/shared/config"
	"github.com/navire-dev/navire/shared/version"
	"github.com/urfave/cli/v3"

	"github.com/navire-dev/navire/navirectl/internal/command"
	clilogger "github.com/navire-dev/navire/navirectl/internal/logger"
	"github.com/navire-dev/navire/navirectl/internal/runtime"
)

// NewRootCmd builds the Navire CLI command tree.
// Command registration remains centralized in commands.go so policies and
// middleware stay visible at the registry boundary.
func NewRootCmd() *cli.Command {
	root := &cli.Command{
		Name:        "navirectl",
		Usage:       "Manage and validate Navire configuration",
		Description: "Navire's command-line interface for declarative configuration and operations.",
		Version:     version.Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-level",
				Value: "info",
				Usage: "Log level: debug, info, warn or error",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			level, ok := sharedconfig.NormalizeLogLevel(cmd.String("log-level"))
			if !ok {
				return ctx, fmt.Errorf("log-level must be one of debug, info, warn or error")
			}

			runtimeContext := runtime.New(nil, cmd.Root().Writer, cmd.Root().ErrWriter)
			runtimeContext.Reader = cmd.Root().Reader
			runtimeContext.Logger = clilogger.New(clilogger.Options{
				Level: level,
				Color: true,
				Out:   cmd.Root().Writer,
				Err:   cmd.Root().ErrWriter,
			})
			command.SetRuntime(cmd.Root(), runtimeContext)
			return ctx, nil
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return fmt.Errorf("a command is required; use %q for available commands", cmd.Root().Name+" --help")
		},
	}

	RegisterSubCommands(root)

	return root
}

// Execute runs the Navire CLI with the supplied context.
func Execute(ctx context.Context, args []string) error {
	return NewRootCmd().Run(ctx, args)
}
