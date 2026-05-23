package service

import (
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"

	"go.opentelemetry.io/otel/metric"
)

const (
	histogramNameDuration      string = "http.server.duration"
	histogramNameResponseBytes string = "http.server.response.bytes"
	histogramNameRequestBytes  string = "http.server.request.bytes"
)

var histogramNames = []string{
	histogramNameDuration,
	histogramNameResponseBytes,
	histogramNameRequestBytes,
}

func createHistogram(meter metrics.Meter, histogramName string) (metrics.Float64Histogram, error) {
	switch histogramName {
	default:
		return nil, errors.Must(errors.New("unsupported histogram name"))
	case histogramNameDuration:
		return meter.Float64Histogram(histogramName,
			metric.WithDescription("determines the aggregate latency of every call"),
			metric.WithUnit("s"),
		)
	case histogramNameResponseBytes:
		return meter.Float64Histogram(histogramName,
			metric.WithDescription("determines the aggregate size of each response"),
			metric.WithUnit("bytes"),
		)
	case histogramNameRequestBytes:
		return meter.Float64Histogram(histogramName,
			metric.WithDescription("determines the aggregate size of each request"),
			metric.WithUnit("bytes"),
		)
	}
}
