package log

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
)

func New(level zerolog.Level) zerolog.Logger {
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	logger := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	return logger
}
func Setup() zerolog.Level {
	level := zerolog.InfoLevel
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		level = zerolog.DebugLevel
	case "error":
		level = zerolog.ErrorLevel
	case "disabled":
		level = zerolog.Disabled
	}
	return level
}

type ctxKey struct{}

func WithLogger(ctx context.Context, l zerolog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

func FromContext(ctx context.Context) zerolog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(zerolog.Logger); ok {
		return l
	}
	return zerolog.Nop()
}
