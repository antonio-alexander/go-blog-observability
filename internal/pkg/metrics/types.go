package metrics

import "go.opentelemetry.io/otel/metric"

type Metrics interface {
	Meter(name string, options ...metric.MeterOption) Meter
}
