package service

import (
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"

	"go.opentelemetry.io/otel/metric"
)

const (
	counterNamePanicsTotal        string = "http.server.panics.total"
	counterNameActiveRequests     string = "http.server.active_requests"
	counterNameRequestsTotal      string = "http.server.requests.total"
	counterNameRequestsSuccessful string = "http.server.requests.success"
	counterNameRequestsFailed     string = "http.server.requests.failure"
	counterNameTotalResponseBytes string = "http.server.response.total_bytes"
	counterNameTotalRequestBytes  string = "http.server.request.total_bytes"
)

var counterNames = []string{
	counterNamePanicsTotal,
	counterNameActiveRequests,
	counterNameRequestsTotal,
	counterNameTotalResponseBytes,
	counterNameTotalRequestBytes,
	counterNameRequestsSuccessful,
	counterNameRequestsFailed,
}

func createCounter(meter metrics.Meter, counterName string) (metrics.Int64Counter, error) {
	switch counterName {
	default:
		return nil, errors.Must(errors.New("unsupported counter name"))
	case counterNamePanicsTotal:
		return meter.Int64UpDownCounter(counterName,
			metric.WithDescription("counts the number of panics"))
	case counterNameActiveRequests:
		return meter.Int64UpDownCounter(counterName,
			metric.WithDescription("counts the number of active requests"))
	case counterNameRequestsTotal:
		return meter.Int64UpDownCounter(counterName,
			metric.WithDescription("counts the total number of requests handled"))
	case counterNameTotalResponseBytes:
		return meter.Int64Counter(counterName,
			metric.WithUnit("bytes"),
			metric.WithDescription("counts the total number of bytes sent"))
	case counterNameTotalRequestBytes:
		return meter.Int64Counter(counterName,
			metric.WithUnit("bytes"),
			metric.WithDescription("counts the total number of bytes received"))
	case counterNameRequestsSuccessful:
		return meter.Int64Counter(counterName,
			metric.WithDescription("counts the total number of successful requests"))
	case counterNameRequestsFailed:
		return meter.Int64Counter(counterName,
			metric.WithDescription("counts the total number of failed requests"))
	}
}
