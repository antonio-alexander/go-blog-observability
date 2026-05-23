package metrics

import (
	"context"

	"github.com/antonio-alexander/go-blog-observability/internal"

	"go.opentelemetry.io/otel/metric"
)

type nullInt64Counter struct{}

func (n *nullInt64Counter) Add(ctx context.Context, incr int64, options ...metric.AddOption) {}

type nullInt64UpDownCounter struct{}

func (n *nullInt64UpDownCounter) Add(ctx context.Context, incr int64, options ...metric.AddOption) {}

type nullInt64Histogram struct{}

func (n *nullInt64Histogram) Record(ctx context.Context, incr int64, options ...metric.RecordOption) {
}

type nullFloat64Histogram struct{}

func (n *nullFloat64Histogram) Record(ctx context.Context, incr float64, options ...metric.RecordOption) {
}

type nullMeter struct{}

func (n *nullMeter) Int64Counter(name string, options ...metric.Int64CounterOption) (Int64Counter, error) {
	return &nullInt64Counter{}, nil
}

func (n *nullMeter) Int64UpDownCounter(name string, options ...metric.Int64UpDownCounterOption) (Int64UpDownCounter, error) {
	return &nullInt64UpDownCounter{}, nil
}

func (n *nullMeter) Int64Histogram(name string, options ...metric.Int64HistogramOption) (Int64Histogram, error) {
	return &nullInt64Histogram{}, nil
}

func (n *nullMeter) Float64Histogram(name string, options ...metric.Float64HistogramOption) (Float64Histogram, error) {
	return &nullFloat64Histogram{}, nil
}

type metricsNull struct{}

func NewNull(parameters ...any) interface {
	internal.Configurer
	internal.Opener
	Metrics
} {
	return &metricsNull{}
}

func (m *metricsNull) Configure(envs map[string]string) error { return nil }

func (m *metricsNull) Open(ctx context.Context) error { return nil }

func (m *metricsNull) Close(ctx context.Context) {}

func (m *metricsNull) Meter(name string, options ...metric.MeterOption) Meter {
	return &nullMeter{}
}
