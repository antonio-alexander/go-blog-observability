package authz

import (
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"

	"go.opentelemetry.io/otel/metric"
)

const (
	counterNameAuthzAccessGranted string = "authz.access.granted"
	counterNameAuthzAccessDenied  string = "authz.access.denied"
)

var counterNames = []string{
	counterNameAuthzAccessGranted,
	counterNameAuthzAccessDenied,
}

func createCounter(meter metrics.Meter, counterName string) (metrics.Int64Counter, error) {
	switch counterName {
	default:
		return nil, errors.Must(errors.New("unsupported counter name"))
	case counterNameAuthzAccessGranted:
		return meter.Int64Counter(counterName,
			metric.WithDescription("counts the number of times access was granted"))
	case counterNameAuthzAccessDenied:
		return meter.Int64Counter(counterName,
			metric.WithDescription("counts the number of times access was denied"))
	}
}
