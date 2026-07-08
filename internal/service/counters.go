package service

import (
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"
)

const (
	counterNameActiveRequests     string = "http_server_requests_in_flight"
	counterNameRequestsTotal      string = "http_server_requests_total"
	counterNameRequestsSuccessful string = "http_server_requests_success"
	counterNameRequestsFailed     string = "http_server_requests_failure"
	counterNameTotalResponseBytes string = "http_server_response_total_bytes"
	counterNameTotalRequestBytes  string = "http_server_request_total_bytes"
	counterNamePanicsTotal        string = "http_server_panics_total"
)

func createCounters(meter metrics.Meter, counterHelper *metrics.Int64CounterHelper) error {
	for _, counterName := range []string{
		counterNamePanicsTotal,
		counterNameRequestsTotal,
		counterNameTotalResponseBytes,
		counterNameTotalRequestBytes,
		counterNameRequestsSuccessful,
		counterNameRequestsFailed,
	} {
		switch counterName {
		default:
			return errors.Must(errors.New("unsupported counter name"))
		case counterNamePanicsTotal:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the number of panics"); err != nil {
				return err
			}
		case counterNameRequestsTotal:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the total number of requests handled"); err != nil {
				return err
			}
		case counterNameTotalResponseBytes:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the total number of bytes sent"); err != nil {
				return err
			}
		case counterNameTotalRequestBytes:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the total number of bytes received"); err != nil {
				return err
			}
		case counterNameRequestsSuccessful:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the total number of successful requests"); err != nil {
				return err
			}
		case counterNameRequestsFailed:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the total number of failed requests"); err != nil {
				return err
			}
		}
	}
	return nil
}

func createUpDownCounters(meter metrics.Meter, counterHelper *metrics.Int64UpDownCounterHelper) error {
	for _, counterName := range []string{
		counterNameActiveRequests,
	} {
		switch counterName {
		default:
			return errors.Must(errors.New("unsupported counter name"))
		case counterNameActiveRequests:
			if err := counterHelper.CreateCounter(meter, counterName,
				"counts the total number of active requests"); err != nil {
				return err
			}
		}
	}
	return nil
}
