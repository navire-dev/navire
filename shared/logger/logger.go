package logger

import (
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	sharedconfig "github.com/navire-dev/navire/shared/config"
)

// Logger is the structured logger used across the project.
// NOTE: Do NOT call Fatal/Fatalf in library code — reserve them for main().
type Logger interface {
	// Structured
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)

	// Formatted
	Debugf(template string, args ...any)
	Infof(template string, args ...any)
	Warnf(template string, args ...any)
	Errorf(template string, args ...any)

	// Contextual
	With(fields ...zap.Field) Logger
	Named(name string) Logger

	// Dynamic level
	SetLevel(level string) error

	// Flush buffers (ignore benign errors on stdio)
	Sync() error
}

type loggerImpl struct {
	base    *zap.Logger
	sugared *zap.SugaredLogger
	level   zap.AtomicLevel // keep atomic level to change at runtime
}

type Options struct {
	Level       string // "debug","info","warn","error"
	Pretty      bool   // console encoder (dev) vs json (prod)
	Service     string // "core" | "dispatch" | etc.
	Version     string // app version
	Commit      string // git sha
	Environment string // "dev" | "prod" | "staging"
}

// New creates a production-ready logger.
func New(level string, pretty bool) Logger {
	return NewWithOptions(Options{Level: level, Pretty: pretty})
}

// NewWithOptions allows richer base fields (service/version/etc.).
func NewWithOptions(opts Options) Logger {
	cfg := zap.NewProductionConfig()
	if opts.Pretty {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		// Timestamps in RFC3339Nano for consistency across services
		cfg.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	}

	lvl := parseLevel(opts.Level)
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.DisableCaller = true

	base, err := cfg.Build(
		// Ordinary errors should stay concise and structured. Stacktraces are
		// reserved for panic/fatal-level failures, not expected HTTP or reload
		// errors.
		zap.AddStacktrace(zapcore.PanicLevel),
	)
	if err != nil {
		panic(err)
	}

	// Attach common fields at the root
	fields := make([]zap.Field, 0, 4)
	if opts.Service != "" {
		fields = append(fields, zap.String("service", opts.Service))
	}
	if opts.Version != "" {
		fields = append(fields, zap.String("version", opts.Version))
	}
	if opts.Commit != "" {
		fields = append(fields, zap.String("commit", opts.Commit))
	}
	if opts.Environment != "" {
		fields = append(fields, zap.String("env", opts.Environment))
	}

	base = base.With(fields...)

	return &loggerImpl{
		base:    base,
		sugared: base.Sugar(),
		level:   cfg.Level,
	}
}

func parseLevel(s string) zapcore.Level {
	normalized, ok := sharedconfig.NormalizeLogLevel(s)
	if !ok {
		return zapcore.InfoLevel
	}
	switch normalized {
	case "debug":
		return zapcore.DebugLevel
	case "info", "":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Structured
func (l *loggerImpl) Debug(msg string, fields ...zap.Field) { l.base.Debug(msg, fields...) }
func (l *loggerImpl) Info(msg string, fields ...zap.Field)  { l.base.Info(msg, fields...) }
func (l *loggerImpl) Warn(msg string, fields ...zap.Field)  { l.base.Warn(msg, fields...) }
func (l *loggerImpl) Error(msg string, fields ...zap.Field) { l.base.Error(msg, fields...) }

// Formatted
func (l *loggerImpl) Debugf(t string, args ...interface{}) { l.sugared.Debugf(t, args...) }
func (l *loggerImpl) Infof(t string, args ...interface{})  { l.sugared.Infof(t, args...) }
func (l *loggerImpl) Warnf(t string, args ...interface{})  { l.sugared.Warnf(t, args...) }
func (l *loggerImpl) Errorf(t string, args ...interface{}) { l.sugared.Errorf(t, args...) }

// Contextual
func (l *loggerImpl) With(fields ...zap.Field) Logger {
	n := l.base.With(fields...)
	return &loggerImpl{
		base:    n,
		sugared: n.Sugar(),
		level:   l.level,
	}
}

func (l *loggerImpl) Named(name string) Logger {
	n := l.base.Named(name)
	return &loggerImpl{
		base:    n,
		sugared: n.Sugar(),
		level:   l.level,
	}
}

// Dynamic level (change at runtime)
func (l *loggerImpl) SetLevel(level string) error {
	normalized, ok := sharedconfig.NormalizeLogLevel(level)
	if !ok {
		return errors.New("invalid log level")
	}
	l.level.SetLevel(parseLevel(normalized))
	return nil
}

// Sync flushes buffers; ignore benign errors from stdio sinks.
func (l *loggerImpl) Sync() error {
	err := l.base.Sync()
	// On some platforms stdio sync returns "invalid argument"; ignore it.
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "invalid argument") {
		return nil
	}
	return err
}

// Field constructors (re-exported from zap for convenience)
// This allows other packages to use structured logging without importing zap directly.
func String(key, val string) zap.Field                 { return zap.String(key, val) }
func Int(key string, val int) zap.Field                { return zap.Int(key, val) }
func Int64(key string, val int64) zap.Field            { return zap.Int64(key, val) }
func Duration(key string, val time.Duration) zap.Field { return zap.Duration(key, val) }
func Error(err error) zap.Field                        { return zap.Error(err) }
