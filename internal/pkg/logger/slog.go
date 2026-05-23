package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	internal "github.com/antonio-alexander/go-blog-observability/internal"
	pkgcontext "github.com/antonio-alexander/go-blog-observability/internal/pkg/context"
)

type loggerSlog struct {
	config struct {
		level         slog.Level
		enableSprintf bool
		serviceName   string
		hostname      string
	}
	*slog.Logger
}

func NewSlog(parameters ...any) interface {
	internal.Configurer
	internal.Opener
	Logger
} {
	return &loggerSlog{}
}

func (l *loggerSlog) Configure(envs map[string]string) error {
	//set defaults
	l.config.level = Error
	l.config.serviceName = "go-blog-observability"
	l.config.hostname = "localhost"

	//configure for envs
	for key, value := range envs {
		switch key {
		case "LOG_LEVEL":
			l.config.level = atoLogLevel(value)
		case "HOSTNAME":
			l.config.hostname = value
		case "SERVICE_NAME":
			l.config.serviceName = value
		}
	}
	return nil
}

func (l *loggerSlog) Open(ctx context.Context) error {
	l.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false,
		Level:     l.config.level,
	})).With(
		slog.String("hostname", l.config.hostname),
		slog.String("service_name", l.config.serviceName),
	)
	return nil
}

func (l *loggerSlog) Close(ctx context.Context) {}

func (l *loggerSlog) Error(ctx context.Context, msg string, items ...any) {
	var attrs []slog.Attr

	if l.config.enableSprintf && hasFormatIdentifiers(msg) {
		msg = fmt.Sprintf(msg, items...)
	}
	if ctxAttrs := pkgcontext.AttributesFrom(ctx); len(ctxAttrs) > 0 {
		attrs = append(attrs, ctxAttrs...)
	}
	attrs = append(attrs, getAttributes(items)...)
	l.LogAttrs(ctx, Error, msg, attrs...)
}

func (l *loggerSlog) Info(ctx context.Context, msg string, items ...any) {
	var attrs []slog.Attr

	if l.config.enableSprintf && hasFormatIdentifiers(msg) {
		msg = fmt.Sprintf(msg, items...)
	}
	if ctxAttrs := pkgcontext.AttributesFrom(ctx); len(ctxAttrs) > 0 {
		attrs = append(attrs, ctxAttrs...)
	}
	attrs = append(attrs, getAttributes(items)...)
	l.LogAttrs(ctx, Info, msg, attrs...)
}

func (l *loggerSlog) Debug(ctx context.Context, msg string, items ...any) {
	var attrs []slog.Attr

	if l.config.enableSprintf && hasFormatIdentifiers(msg) {
		msg = fmt.Sprintf(msg, items...)
	}
	if ctxAttrs := pkgcontext.AttributesFrom(ctx); len(ctxAttrs) > 0 {
		attrs = append(attrs, ctxAttrs...)
	}
	attrs = append(attrs, getAttributes(items)...)
	l.LogAttrs(ctx, Debug, msg, attrs...)
}

func (l *loggerSlog) Warn(ctx context.Context, msg string, items ...any) {
	var attrs []slog.Attr

	if l.config.enableSprintf && hasFormatIdentifiers(msg) {
		msg = fmt.Sprintf(msg, items...)
	}
	if ctxAttrs := pkgcontext.AttributesFrom(ctx); len(ctxAttrs) > 0 {
		attrs = append(attrs, ctxAttrs...)
	}
	attrs = append(attrs, getAttributes(items)...)
	l.LogAttrs(ctx, Warn, msg, attrs...)
}
