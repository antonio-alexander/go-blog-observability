package service

import (
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"
)

const (
	histogramNameDuration      string = "http_server_request_duration"
	histogramNameRequestBytes  string = "http_server_request_bytes"
	histogramNameResponseBytes string = "http_server_response_bytes"
)

func createHistograms(meter metrics.Meter, float64Helper *metrics.Float64HistogramHelper, int64Helper *metrics.Int64HistogramHelper) error {
	for _, histogramName := range []string{
		histogramNameDuration,
		histogramNameResponseBytes,
		histogramNameRequestBytes,
	} {
		switch histogramName {
		default:
			return errors.Must(errors.New("unsupported histogram name"))
		case histogramNameDuration:
			if err := float64Helper.CreateHistogram(meter, histogramName,
				"determines the aggregate latency of each request"); err != nil {
				return err
			}
			// 	metric.WithDescription("determines the aggregate latency of every call"),
			// 	metric.WithUnit("s"),
		case histogramNameResponseBytes:
			if err := int64Helper.CreateHistogram(meter, histogramName,
				"determines the aggregate size of each response"); err != nil {
				return err
			}
			// 	metric.WithDescription("determines the aggregate size of each response"),
			// 	metric.WithUnit("bytes"),
		case histogramNameRequestBytes:
			if err := int64Helper.CreateHistogram(meter, histogramName,
				"determines the aggregate size of each request"); err != nil {
				return err
			}
			// metric.WithDescription("determines the aggregate size of each request"),
			// metric.WithUnit("bytes"),
		}
	}
	return nil
}
