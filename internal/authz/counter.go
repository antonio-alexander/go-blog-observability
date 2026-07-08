package authz

import (
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"
)

const (
	counterNameAuthzAccessGranted string = "authz_access_granted"
	counterNameAuthzAccessDenied  string = "authz_access_denied"
)

func createCounters(meter metrics.Meter, counterHelper *metrics.Int64CounterHelper) error {
	for _, counterName := range []string{
		counterNameAuthzAccessGranted,
		counterNameAuthzAccessDenied,
	} {
		switch counterName {
		default:
			return errors.Must(errors.New("unsupported counter name"))
		case counterNameAuthzAccessGranted:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the number of times access was granted"); err != nil {
				return err
			}
		case counterNameAuthzAccessDenied:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the number of times access was denied"); err != nil {
				return err
			}
		}
	}
	return nil
}
