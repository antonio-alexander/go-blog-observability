package logic

import (
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"

	"go.opentelemetry.io/otel/metric"
)

const (
	counterNameEmployeeCacheHit  string = "logic.employee.hit"
	counterNameEmployeeCacheMiss string = "logic.employee.miss"
	counterNameSleepCacheHit     string = "logic.sleep.hit"
	counterNameSleepCacheMiss    string = "logic.sleep.miss"
)

var counterNames = []string{
	counterNameEmployeeCacheHit,
	counterNameEmployeeCacheMiss,
	counterNameSleepCacheHit,
	counterNameSleepCacheMiss,
}

func createCounter(meter metrics.Meter, counterName string) (metrics.Int64Counter, error) {
	switch counterName {
	default:
		return nil, errors.Must(errors.New("unsupported counter name"))
	case counterNameEmployeeCacheHit:
		return meter.Int64Counter(counterName,
			metric.WithDescription("counts the number of employee cache hits"))
	case counterNameEmployeeCacheMiss:
		return meter.Int64Counter(counterName,
			metric.WithDescription("counts the number of employee cache misses"))
	case counterNameSleepCacheHit:
		return meter.Int64Counter(counterName,
			metric.WithDescription("counts the number of sleep cache hits"))
	case counterNameSleepCacheMiss:
		return meter.Int64Counter(counterName,
			metric.WithDescription("counts the number of sleep cache misses"))
	}
}
