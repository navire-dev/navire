package middleware

import (
	"context"
	"reflect"
	"testing"

	"github.com/navire-dev/navire/navirectl/internal/command"
	"github.com/navire-dev/navire/navirectl/internal/runtime"
	"github.com/urfave/cli/v3"
)

func TestUseMiddlewareChainPreservesDeclarationOrder(t *testing.T) {
	var calls []string

	trace := func(name string) Middleware {
		return func(next Handler) Handler {
			return func(ctx context.Context, rt *runtime.Context) error {
				calls = append(calls, name+":before")
				err := next(ctx, rt)
				calls = append(calls, name+":after")
				return err
			}
		}
	}

	factory := UseMiddlewareChain(
		trace("first"),
		trace("second"),
	)(func() *cli.Command {
		return command.New("test", "", func(context.Context, *runtime.Context) error {
			calls = append(calls, "action")
			return nil
		})
	})

	cmd := factory()
	command.SetRuntime(cmd, runtime.New(nil, nil, nil))
	if err := cmd.Run(context.Background(), []string{"test"}); err != nil {
		t.Fatalf("run command: %v", err)
	}

	want := []string{"first:before", "second:before", "action", "second:after", "first:after"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("middleware calls = %v, want %v", calls, want)
	}
}

func TestUseMiddlewareChainKeepsNilActionSafe(t *testing.T) {
	factory := UseMiddlewareChain()(func() *cli.Command {
		return command.New("test", "", nil)
	})

	cmd := factory()
	command.SetRuntime(cmd, runtime.New(nil, nil, nil))
	if err := cmd.Run(context.Background(), []string{"test"}); err != nil {
		t.Fatalf("run command: %v", err)
	}
}
