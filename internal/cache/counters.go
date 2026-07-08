package cache

import (
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"
)

const (
	counterNameEmployeeCacheHit                  string = "cache_employee_hit"
	counterNameEmployeeCacheMiss                 string = "cache_employee_miss"
	counterNameEmployeeCacheInProgressSet        string = "cache_employee_in_progress_set"
	counterNameEmployeeCacheInProgressAlreadySet string = "cache_employee_already_set"
	counterNameSleepCacheHit                     string = "cache_sleep_hit"
	counterNameSleepCacheMiss                    string = "cache_sleep_miss"
	counterNameSleepCacheInProgressSet           string = "cache_sleep_in_progress_set"
	counterNameSleepCacheInProgressAlreadySet    string = "cache_sleep_already_set"
)

func createCounters(meter metrics.Meter, counterHelper *metrics.Int64CounterHelper) error {
	for _, counterName := range []string{
		counterNameEmployeeCacheHit,
		counterNameEmployeeCacheMiss,
		counterNameEmployeeCacheInProgressSet,
		counterNameEmployeeCacheInProgressAlreadySet,
		counterNameSleepCacheHit,
		counterNameSleepCacheMiss,
		counterNameSleepCacheInProgressSet,
		counterNameSleepCacheInProgressAlreadySet,
	} {
		switch counterName {
		default:
			return errors.Must(errors.New("unsupported counter name"))
		case counterNameEmployeeCacheHit:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the number of employee cache hits"); err != nil {
				return err
			}
		case counterNameEmployeeCacheMiss:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the number of employee cache misses"); err != nil {
				return err
			}
		case counterNameEmployeeCacheInProgressSet:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the number of employee cache in progress set"); err != nil {
				return err
			}
		case counterNameEmployeeCacheInProgressAlreadySet:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the number of employee cache in progress already set"); err != nil {
				return err
			}
		case counterNameSleepCacheHit:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the number of sleep cache hits"); err != nil {
				return err
			}
		case counterNameSleepCacheMiss:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the number of sleep cache misses"); err != nil {
				return err
			}
		case counterNameSleepCacheInProgressSet:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the number of sleep cache in progress set"); err != nil {
				return err
			}
		case counterNameSleepCacheInProgressAlreadySet:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the number of sleep cache in progress already set"); err != nil {
				return err
			}
		}
	}
	return nil
}
