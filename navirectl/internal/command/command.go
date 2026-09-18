package command

import (
	"context"
	"fmt"

	"github.com/navire-dev/navire/navirectl/internal/runtime"
	"github.com/urfave/cli/v3"
)

const (
	runtimeKey = "navirectl.runtime"
	handlerKey = "navirectl.handler"
)

// Binder copies framework-specific command values into the runtime context.
type Binder func(*cli.Command, *runtime.Context)

// BindBool exposes a boolean flag to a framework-independent handler.
func BindBool(name string) Binder {
	return func(cmd *cli.Command, rt *runtime.Context) {
		rt.Values[name] = cmd.Bool(name)
	}
}

// New creates a command whose implementation receives the explicit runtime
// context instead of accessing urfave or the root command directly.
func New(name, usage string, handler runtime.Handler, binders ...Binder) *cli.Command {
	cmd := &cli.Command{
		Name:  name,
		Usage: usage,
		Metadata: map[string]any{
			handlerKey: handler,
		},
	}
	cmd.Action = func(ctx context.Context, cmd *cli.Command) error {
		base, err := Runtime(cmd)
		if err != nil {
			return err
		}
		rt := base.Clone()
		for _, binder := range binders {
			binder(cmd, rt)
		}
		registered, err := Handler(cmd)
		if err != nil || registered == nil {
			return err
		}
		return registered(ctx, rt)
	}
	return cmd
}

// SetRuntime attaches the explicit runtime to the command tree.
func SetRuntime(cmd *cli.Command, rt *runtime.Context) {
	if cmd.Metadata == nil {
		cmd.Metadata = make(map[string]any)
	}
	cmd.Metadata[runtimeKey] = rt
}

// Runtime returns the runtime attached to the root command.
func Runtime(cmd *cli.Command) (*runtime.Context, error) {
	value, ok := cmd.Root().Metadata[runtimeKey]
	if !ok {
		return nil, fmt.Errorf("CLI runtime context is not initialized")
	}
	rt, ok := value.(*runtime.Context)
	if !ok || rt == nil {
		return nil, fmt.Errorf("CLI runtime context has an invalid type")
	}
	return rt, nil
}

// Handler returns the framework-independent handler registered on a command.
func Handler(cmd *cli.Command) (runtime.Handler, error) {
	value, ok := cmd.Metadata[handlerKey]
	if !ok {
		return nil, fmt.Errorf("command %q has no handler", cmd.Name)
	}
	if value == nil {
		return nil, nil
	}
	handler, ok := value.(runtime.Handler)
	if !ok {
		return nil, fmt.Errorf("command %q has an invalid handler", cmd.Name)
	}
	return handler, nil
}

// SetHandler replaces the framework-independent handler on a command.
func SetHandler(cmd *cli.Command, handler runtime.Handler) {
	if cmd.Metadata == nil {
		cmd.Metadata = make(map[string]any)
	}
	cmd.Metadata[handlerKey] = handler
}
