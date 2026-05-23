package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal"
	pkgcontext "github.com/antonio-alexander/go-blog-observability/internal/pkg/context"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

type loggerOpenTelemetry struct {
	config struct {
		level              slog.Level
		serviceName        string
		hostname           string
		otelProtocol       string
		otelEndpoint       string
		otelExportInterval time.Duration
		enableSprintf      bool
	}
	loggerOpenTelemetryProvider *sdklog.LoggerProvider
	*slog.Logger
}

func NewOpenTelemetry(parameters ...any) interface {
	internal.Configurer
	internal.Opener
	Logger
} {
	return &loggerOpenTelemetry{}
}

func (l *loggerOpenTelemetry) Configure(envs map[string]string) error {
	//set defaults
	l.config.level = Error
	l.config.otelProtocol = "http"
	l.config.otelEndpoint = "localhost:4318"
	l.config.otelExportInterval = time.Second
	l.config.serviceName = "go-blog-observability"
	l.config.hostname = "localhost"

	//configure for envs
	for key, value := range envs {
		switch key {
		case "OTEL_LOG_LEVEL", "LOG_LEVEL":
			l.config.level = atoLogLevel(value)
		case "OTEL_PROTOCOL":
			l.config.otelProtocol = value
		case "OTEL_HOST", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT":
			l.config.otelEndpoint = value
		case "OTEL_EXPORT_INTERVAL":
			i, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			l.config.otelExportInterval = time.Duration(i) * time.Second
		case "HOSTNAME":
			l.config.hostname = value
		case "SERVICE_NAME":
			l.config.serviceName = value
		}
	}
	return nil
}

func (l *loggerOpenTelemetry) Open(ctx context.Context) error {
	//create exporter and processor for standard out
	stdoutExporter, err := stdoutlog.New(stdoutlog.WithWriter(os.Stdout))
	if err != nil {
		return err
	}
	// stdoutProcessor := sdklog.NewBatchProcessor(stdoutExporter)
	stdoutProcessor := newFilterProcessor(
		sdklog.NewBatchProcessor(stdoutExporter), l.config.level)

	//create exporter and processor for open telemetry
	httpOpts := []otlploghttp.Option{}
	if l.config.otelProtocol == "http" {
		httpOpts = append(httpOpts, otlploghttp.WithInsecure())
	}
	httpOpts = append(httpOpts,
		otlploghttp.WithEndpoint(l.config.otelEndpoint))
	otelExporter, err := otlploghttp.New(ctx, httpOpts...)
	if err != nil {
		return err
	}
	otelProcessor := sdklog.NewBatchProcessor(otelExporter,
		sdklog.WithExportInterval(l.config.otelExportInterval))

	// create resource
	resource, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(l.config.serviceName),
			semconv.HostNameKey.String(l.config.hostname),
		),
	)
	if err != nil {
		return err
	}

	//create loggerOpenTelemetry provider
	l.loggerOpenTelemetryProvider = sdklog.NewLoggerProvider(
		sdklog.WithProcessor(otelProcessor),
		sdklog.WithProcessor(stdoutProcessor),
		sdklog.WithResource(resource),
	)

	//create handler and store loggerOpenTelemetry
	otelHandler := otelslog.NewHandler(l.config.serviceName,
		otelslog.WithLoggerProvider(l.loggerOpenTelemetryProvider),
	)
	l.Logger = slog.New(otelHandler)
	return nil
}

func (l *loggerOpenTelemetry) Close(ctx context.Context) {
	if err := l.loggerOpenTelemetryProvider.Shutdown(ctx); err != nil {
		fmt.Println(err)
	}
	l.loggerOpenTelemetryProvider = nil
}

func (l *loggerOpenTelemetry) Error(ctx context.Context, msg string, items ...any) {
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

func (l *loggerOpenTelemetry) Info(ctx context.Context, msg string, items ...any) {
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

func (l *loggerOpenTelemetry) Debug(ctx context.Context, msg string, items ...any) {
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

func (l *loggerOpenTelemetry) Warn(ctx context.Context, msg string, items ...any) {
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
