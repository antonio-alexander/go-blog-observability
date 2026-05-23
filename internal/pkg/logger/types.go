package logger

import (
	"context"
	"log/slog"
	"strings"
)

const (
	Debug slog.Level = slog.LevelDebug
	Info  slog.Level = slog.LevelInfo
	Warn  slog.Level = slog.LevelWarn
	Error slog.Level = slog.LevelError
)

type Logger interface {
	Debug(ctx context.Context, msg string, items ...any)
	Info(ctx context.Context, msg string, items ...any)
	Error(ctx context.Context, msg string, items ...any)
	Warn(ctx context.Context, msg string, items ...any)
}

func atoLogLevel(a string) slog.Level {
	switch strings.ToLower(a) {
	default:
		return 0
	case "error":
		return Error
	case "info":
		return Info
	case "debug":
		return Debug
	}
}

type Loggable interface {
	GetAttributes() []slog.Attr
}
