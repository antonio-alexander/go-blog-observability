package tracer

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal"
	pkgcontext "github.com/antonio-alexander/go-blog-observability/internal/pkg/context"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

type tracerOpenTelemetry struct {
	logger.Logger
	config struct {
		serviceName        string
		hostname           string
		otelProtocol       string
		otelEndpoint       string
		otelExportInterval time.Duration
	}
	tracer         trace.Tracer
	tracerProvider *sdktrace.TracerProvider
	propagator     propagation.TextMapPropagator
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
	m.config.otelProtocol = "http"
	m.config.otelEndpoint = "localhost:4318"
	m.config.hostname = "localhost"
	m.config.otelExportInterval = time.Second

	//set configuration
	for key, value := range envs {
		switch key {
		case "OTEL_SERVICE_NAME", "SERVICE_NAME":
			m.config.serviceName = value
		case "HOSTNAME":
			m.config.hostname = value
		case "OTEL_PROTOCOL":
			m.config.otelProtocol = value
		case "OTEL_HOST", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT":
			m.config.otelEndpoint = value
		case "OTEL_EXPORT_INTERVAL":
			i, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			m.config.otelExportInterval = time.Duration(i) * time.Second
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

	//create propagator to communicate traces via REST
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // Handles traceparent and tracestate headers
		propagation.Baggage{},      // Handles custom key-value metadata
	)
	m.propagator = propagator

	//create trace provider and tracer
	m.tracerProvider = sdktrace.NewTracerProvider(tracerProviderOpts...)
	m.tracer = m.tracerProvider.Tracer(m.config.serviceName)
	return nil
}

func (m *tracerOpenTelemetry) Close(ctx context.Context) {
	if err := m.tracerProvider.Shutdown(ctx); err != nil {
		m.Error(ctx, "unable to shutdown tracer provider",
			err)
	}
}

func (m *tracerOpenTelemetry) Start(ctx context.Context, spanName string,
	opts ...trace.SpanStartOption) (context.Context, Span) {
	opts = append(opts,
		trace.WithAttributes(pkgcontext.ToAttributeKeyValue(ctx)...))
	ctx, span := m.tracer.Start(ctx, spanName, opts...)
	return ctx, &openTelemetrySpan{
		Span:    span,
		Logger:  m.Logger,
		Context: ctx,
	}
}

func (m *tracerOpenTelemetry) TracerProvider() *sdktrace.TracerProvider {
	return m.tracerProvider
}

func (m *tracerOpenTelemetry) PropagateTrace(ctx context.Context, r *http.Request) {
	m.propagator.Inject(ctx, propagation.HeaderCarrier(r.Header))
}

func (m *tracerOpenTelemetry) ExtractTrace(r *http.Request) context.Context {
	return m.propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
}

type openTelemetrySpan struct {
	trace.Span
	logger.Logger
	context.Context
}

func (s *openTelemetrySpan) End(options ...trace.SpanEndOption) {
	s.Span.End(options...)
	s.Trace(s.Context, "trace ended")
}
