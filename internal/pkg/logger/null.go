package logger

import (
	"context"

	internal "github.com/antonio-alexander/go-blog-observability/internal"
)

type loggerNull struct{}

func NewNull(parameters ...any) interface {
	internal.Configurer
	internal.Opener
	Logger
} {
	return &loggerNull{}
}

func (l *loggerNull) Configure(envs map[string]string) error { return nil }

func (l *loggerNull) Open(ctx context.Context) error { return nil }

func (l *loggerNull) Close(ctx context.Context) {}

func (l *loggerNull) Error(ctx context.Context, msg string, items ...any) {}

func (l *loggerNull) Info(ctx context.Context, msg string, items ...any) {}

func (l *loggerNull) Debug(ctx context.Context, msg string, items ...any) {}

func (l *loggerNull) Warn(ctx context.Context, msg string, items ...any) {}
