package runtime

import (
	"context"
	"io"
)

// Context carries command execution data explicitly between CLI layers.
// It is intentionally separate from context.Context, which remains reserved
// for cancellation, deadlines and execution-scoped values.
type Context struct {
	Args      []string
	Reader    io.Reader
	Writer    io.Writer
	ErrWriter io.Writer
	Logger    Logger
	Values    map[string]any
}

// Logger is the logging surface required by the CLI runtime.
type Logger interface {
	Info(message string, args ...any)
	Success(message string, args ...any)
	Warn(message string, args ...any)
	Error(message string, args ...any)
	Debug(message string, args ...any)
}

// Handler is the framework-independent shape used by commands and
// middleware.
type Handler func(context.Context, *Context) error

// New creates a runtime context and copies the argument slice so callers
// cannot mutate the command state accidentally.
func New(args []string, writer, errWriter io.Writer) *Context {
	argsCopy := append([]string(nil), args...)
	return &Context{
		Args:      argsCopy,
		Writer:    writer,
		ErrWriter: errWriter,
		Values:    make(map[string]any),
	}
}

// Clone returns an execution-local copy of the runtime context.
func (c *Context) Clone() *Context {
	if c == nil {
		return nil
	}
	clone := *c
	clone.Args = append([]string(nil), c.Args...)
	clone.Values = make(map[string]any, len(c.Values))
	for key, value := range c.Values {
		clone.Values[key] = value
	}
	return &clone
}

// Bool returns a boolean value bound by a command adapter.
func (c *Context) Bool(name string) bool {
	value, ok := c.Values[name].(bool)
	return ok && value
}

func (c *Context) Value(name string) (any, bool) {
	value, ok := c.Values[name]
	return value, ok
}
