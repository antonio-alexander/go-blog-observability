package metrics

import (
	"context"
	"strconv"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type metricsOpenTelemetry struct {
	config struct {
		otelProtocol         string
		otelEndpoint         string
		enableGoMetrics      bool
		enableStdout         bool
		goMetricStatInterval time.Duration
		exportInterval       time.Duration
		serviceName          string
		hostname             string
	}
	*sdkmetric.MeterProvider
	logger.Logger
}

func NewOpenTelemetry(parameters ...any) interface {
	internal.Configurer
	internal.Opener
	Metrics
} {
	m := &metricsOpenTelemetry{}
	for _, parameter := range parameters {
		switch p := parameter.(type) {
		case logger.Logger:
			m.Logger = p
		}
	}
	return m
}

func (m *metricsOpenTelemetry) Configure(envs map[string]string) error {
	//set configuration defaults
	m.config.serviceName = "go-blog-observability"
	m.config.enableGoMetrics = true
	m.config.enableStdout = false
	m.config.exportInterval = time.Second
	m.config.goMetricStatInterval = time.Minute
	m.config.otelProtocol = "http"
	m.config.otelEndpoint = "localhost:4318"
	m.config.hostname = "localhost"

	//get configuration
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
		case "METRICS_ENABLE_STDOUT":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return err
			}
			m.config.enableStdout = b
		case "ENABLE_GO_METRICS":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return err
			}
			m.config.enableGoMetrics = b
		case "METRIC_STAT_INTERVAL":
			i, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			m.config.goMetricStatInterval = time.Duration(i) * time.Second
		case "METRIC_EXPORT_INTERVAL":
			i, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			m.config.exportInterval = time.Duration(i) * time.Second
		}
	}
	return nil
}

func (m *metricsOpenTelemetry) Open(ctx context.Context) error {
	var metricOptions []sdkmetric.Option

	//create open telemetry exporter
	httpOpts := []otlpmetrichttp.Option{}
	if m.config.otelProtocol == "http" {
		httpOpts = append(httpOpts, otlpmetrichttp.WithInsecure())
	}
	httpOpts = append(httpOpts,
		otlpmetrichttp.WithEndpoint(m.config.otelEndpoint))
	otelExporter, err := otlpmetrichttp.New(ctx, httpOpts...)
	if err != nil {
		return err
	}
	metricOptions = append(metricOptions,
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(otelExporter,
			sdkmetric.WithInterval(m.config.exportInterval))),
	)

	//create standard out exporter
	if m.config.enableStdout {
		stdOutExporter, err := stdoutmetric.New()
		if err != nil {
			return err
		}
		metricOptions = append(metricOptions,
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(stdOutExporter)),
		)
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
	metricOptions = append(metricOptions, sdkmetric.WithResource(resource))

	//create meter provider
	meterProvider := sdkmetric.NewMeterProvider(metricOptions...)
	otel.SetMeterProvider(meterProvider)

	//start runtime to capture go metrics
	if m.config.enableGoMetrics {
		if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(m.config.goMetricStatInterval),
			runtime.WithMeterProvider(meterProvider)); err != nil {
			return err
		}
	}
	m.MeterProvider = meterProvider
	return nil
}

func (m *metricsOpenTelemetry) Close(ctx context.Context) {
	if err := m.MeterProvider.Shutdown(ctx); err != nil {
		m.Error(ctx, "error while shutting down meter provider",
			err)
	}
}

func (m *metricsOpenTelemetry) Meter(name string, options ...metric.MeterOption) Meter {
	return &openTelemetryMeter{
		Meter: m.MeterProvider.Meter(name, options...),
	}
}

type openTelemetryInt64Counter struct {
	metric.Int64Counter
}

type openTelemetryInt64UpDownCounter struct {
	metric.Int64UpDownCounter
}

type openTelemetryInt64Histogram struct {
	metric.Int64Histogram
}

type openTelemetryFloat64Histogram struct {
	metric.Float64Histogram
}

type openTelemetryMeter struct {
	metric.Meter
}

func (o *openTelemetryMeter) Int64Counter(name string, options ...metric.Int64CounterOption) (Int64Counter, error) {
	int64Counter, err := o.Meter.Int64Counter(name, options...)
	if err != nil {
		return nil, err
	}
	return &openTelemetryInt64Counter{
		Int64Counter: int64Counter,
	}, nil
}

func (o *openTelemetryMeter) Int64UpDownCounter(name string, options ...metric.Int64UpDownCounterOption) (Int64UpDownCounter, error) {
	int64UpDownCounter, err := o.Meter.Int64UpDownCounter(name, options...)
	if err != nil {
		return nil, err
	}
	return &openTelemetryInt64UpDownCounter{
		Int64UpDownCounter: int64UpDownCounter,
	}, nil
}

func (o *openTelemetryMeter) Int64Histogram(name string, options ...metric.Int64HistogramOption) (Int64Histogram, error) {
	int64Histogram, err := o.Meter.Int64Histogram(name, options...)
	if err != nil {
		return nil, err
	}
	return &openTelemetryInt64Histogram{
		Int64Histogram: int64Histogram,
	}, nil
}

func (o *openTelemetryMeter) Float64Histogram(name string, options ...metric.Float64HistogramOption) (Float64Histogram, error) {
	float64Histogram, err := o.Meter.Float64Histogram(name, options...)
	if err != nil {
		return nil, err
	}
	return &openTelemetryFloat64Histogram{
		Float64Histogram: float64Histogram,
	}, nil
}
