package metrics

import (
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Metrics interface {
	Meter(name string, options ...metric.MeterOption) Meter
	MeterProvider() *sdkmetric.MeterProvider
}
