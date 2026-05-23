package tracer

import (
	"context"
	"strconv"

	"github.com/antonio-alexander/go-blog-observability/internal"
	pkgcontext "github.com/antonio-alexander/go-blog-observability/internal/pkg/context"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

type tracerOpenTelemetry struct {
	config struct {
		serviceName  string
		hostname     string
		enableStdout bool
		otelProtocol string
		otelEndpoint string
	}
	*sdktrace.TracerProvider
	trace.Tracer
	logger.Logger
}

func NewOpenTelemetry(parameters ...any) interface {
	internal.Configurer
	internal.Opener
	Tracer
} {
	m := &tracerOpenTelemetry{}
	for _, parameter := range parameters {
		switch p := parameter.(type) {
		case logger.Logger:
			m.Logger = p
		}
	}
	return m
}

func (m *tracerOpenTelemetry) Configure(envs map[string]string) error {
	//set configuration defaults
	m.config.serviceName = "go-blog-observability"
	m.config.enableStdout = false
	m.config.otelProtocol = "http"
	m.config.otelEndpoint = "localhost:4318"
	m.config.hostname = "localhost"

	//set configuration
	for key, value := range envs {
		switch key {
		case "OTEL_SERVICE_NAME", "SERVICE_NAME":
			m.config.serviceName = value
		case "HOSTNAME":
			m.config.hostname = value
		case "TRACE_ENABLE_STDOUT":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return err
			}
			m.config.enableStdout = b
		case "OTEL_PROTOCOL":
			m.config.otelProtocol = value
		case "OTEL_HOST", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT":
			m.config.otelEndpoint = value
		}
	}
	return nil
}

func (m *tracerOpenTelemetry) Open(ctx context.Context) error {
	var tracerProviderOpts []sdktrace.TracerProviderOption

	//create open telemetry exporter
	httpOpts := []otlptracehttp.Option{}
	if m.config.otelProtocol == "http" {
		httpOpts = append(httpOpts, otlptracehttp.WithInsecure())
	}
	httpOpts = append(httpOpts,
		otlptracehttp.WithEndpoint(m.config.otelEndpoint))
	otelExporter, err := otlptracehttp.New(ctx, httpOpts...)
	if err != nil {
		return err
	}
	tracerProviderOpts = append(tracerProviderOpts,
		sdktrace.WithBatcher(otelExporter))

	//create standard out exporter if configured
	if m.config.enableStdout {
		stdoutExporter, err := stdouttrace.New()
		if err != nil {
			return err
		}
		tracerProviderOpts = append(tracerProviderOpts,
			sdktrace.WithBatcher(stdoutExporter))
	}

	// create resource
	resource, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(m.config.serviceName),
			semconv.HostNameKey.String(m.config.hostname),
		),
	)
	if err != nil {
		return err
	}
	tracerProviderOpts = append(tracerProviderOpts, sdktrace.WithResource(resource))

	//create trace provider and tracer
	m.TracerProvider = sdktrace.NewTracerProvider(tracerProviderOpts...)
	m.Tracer = m.TracerProvider.Tracer(m.config.serviceName)
	otel.SetTracerProvider(m.TracerProvider)
	return nil
}

func (m *tracerOpenTelemetry) Close(ctx context.Context) {
	if err := m.TracerProvider.Shutdown(ctx); err != nil {
		m.Error(ctx, "unable to shutdown tracer provider",
			err)
	}
}

func (m *tracerOpenTelemetry) Start(ctx context.Context, spanName string,
	opts ...trace.SpanStartOption) (context.Context, Span) {
	correlationId := pkgcontext.CorrelationIdFrom(ctx)
	requestId := pkgcontext.RequestIdFrom(ctx)
	opts = append(opts, trace.WithAttributes(
		attribute.String("correlation_id", correlationId),
		attribute.String("request_id", requestId),
	))
	ctx, span := m.Tracer.Start(ctx, spanName, opts...)
	//
	// span.SpanContext().TraceID()
	//
	return ctx, span
}
