package logger

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/navire-dev/navire/navirectl/internal/printer"
	sharedlogger "github.com/navire-dev/navire/shared/logger"
)

type Options struct {
	Level string
	Color bool
	Out   io.Writer
	Err   io.Writer
}

// Logger renders human-readable CLI diagnostics while keeping the shared
// logger available for structured output and future JSON support.
type Logger struct {
	level   string
	out     io.Writer
	errOut  io.Writer
	printer *printer.ColorPrinter
	shared  sharedlogger.Logger
}

func New(opts Options) *Logger {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Err == nil {
		opts.Err = os.Stderr
	}
	if opts.Level == "" {
		opts.Level = "info"
	}

	return &Logger{
		level:   strings.ToLower(opts.Level),
		out:     opts.Out,
		errOut:  opts.Err,
		printer: printer.NewColorPrinter(opts.Color),
		shared:  sharedlogger.New(opts.Level, false),
	}
}

func (l *Logger) Info(message string, args ...any) {
	if l.enabled("info") {
		l.write(l.out, l.printer.Info(message, args...))
	}
}

func (l *Logger) Success(message string, args ...any) {
	if l.enabled("info") {
		l.write(l.out, l.printer.Success(message, args...))
	}
}

func (l *Logger) Warn(message string, args ...any) {
	if l.enabled("warn") {
		l.write(l.errOut, l.printer.Warning(message, args...))
	}
}

func (l *Logger) Error(message string, args ...any) {
	if l.enabled("error") {
		l.write(l.errOut, l.printer.Error(message, args...))
	}
}

func (l *Logger) Debug(message string, args ...any) {
	if l.enabled("debug") {
		l.write(l.errOut, l.printer.Debug(message, args...))
	}
}

func (l *Logger) Shared() sharedlogger.Logger {
	return l.shared
}

func (l *Logger) enabled(level string) bool {
	weight := map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}
	return weight[level] >= weight[l.level]
}

func (l *Logger) write(out io.Writer, message string) {
	_, _ = fmt.Fprintln(out, message)
}
