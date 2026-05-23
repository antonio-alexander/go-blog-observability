package policy

import (
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"

	"go.opentelemetry.io/otel/metric"
)

const (
	histogramNameEvaluationDuration  string = "policy.evaluation.duration"
	histogramNameCompilationDuration string = "policy.compilation.duration"
)

var histogramNames = []string{
	histogramNameEvaluationDuration,
	histogramNameCompilationDuration,
}

func createHistogram(meter metrics.Meter, histogramName string) (metrics.Float64Histogram, error) {
	switch histogramName {
	default:
		return nil, errors.Must(errors.New("unsupported histogram name"))
	case histogramNameEvaluationDuration:
		return meter.Float64Histogram(histogramName,
			metric.WithDescription("determines the aggregate latency of every evaluation"),
			metric.WithUnit("s"),
		)
	case histogramNameCompilationDuration:
		return meter.Float64Histogram(histogramName,
			metric.WithDescription("determines the aggregate latency of every compilation"),
			metric.WithUnit("s"),
		)
	}
}
