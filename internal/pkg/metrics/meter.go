package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

type Int64Counter interface {
	Add(ctx context.Context, incr int64, options ...metric.AddOption)
}

type Int64UpDownCounter interface {
	Add(ctx context.Context, incr int64, options ...metric.AddOption)
}

type Int64Histogram interface {
	Record(ctx context.Context, incr int64, options ...metric.RecordOption)
}

type Float64Histogram interface {
	Record(ctx context.Context, incr float64, options ...metric.RecordOption)
}

type Meter interface {
	Int64Counter(name string, options ...metric.Int64CounterOption) (Int64Counter, error)
	Int64UpDownCounter(name string, options ...metric.Int64UpDownCounterOption) (Int64UpDownCounter, error)
	Int64Histogram(name string, options ...metric.Int64HistogramOption) (Int64Histogram, error)
	Float64Histogram(name string, options ...metric.Float64HistogramOption) (Float64Histogram, error)
}
