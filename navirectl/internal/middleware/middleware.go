package middleware

import (
	"github.com/navire-dev/navire/navirectl/internal/command"
	"github.com/navire-dev/navire/navirectl/internal/runtime"
	"github.com/urfave/cli/v3"
)

// Handler is the framework-independent shape used by CLI middleware.
type Handler = runtime.Handler

// Middleware decorates a command handler with a pre/post policy.
type Middleware func(Handler) Handler

// CommandFactory creates one command instance for registration.
type CommandFactory func() *cli.Command

// MiddlewareChain decorates a command factory with middleware.
type MiddlewareChain func(CommandFactory) CommandFactory

// UseMiddlewareChain applies middleware in declaration order.
//
// UseMiddlewareChain(first, second)(factory) executes as:
// first -> second -> command action.
func UseMiddlewareChain(middlewares ...Middleware) MiddlewareChain {
	return func(factory CommandFactory) CommandFactory {
		return func() *cli.Command {
			cmd := factory()
			apply(cmd, middlewares)
			return cmd
		}
	}
}

func apply(cmd *cli.Command, middlewares []Middleware) {
	if action, err := command.Handler(cmd); err == nil && action != nil {
		for i := len(middlewares) - 1; i >= 0; i-- {
			action = middlewares[i](action)
		}
		command.SetHandler(cmd, action)
	}
	for _, child := range cmd.Commands {
		apply(child, middlewares)
	}
}
