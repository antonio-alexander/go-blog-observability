package metrics_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"
)

var envs = map[string]string{
	"OTEL_LOG_LEVEL":        "info",
	"METRICS_ENABLE_STDOUT": "true",
}

func init() {
	envs := make(map[string]string)
	for _, env := range os.Environ() {
		if key, value, ok := strings.Cut(env, "="); ok && value != "" {
			envs[key] = value
		}
	}
}

func TestXxx(t *testing.T) {
	ctx := t.Context()

	//setup logger
	logger := logger.NewSlog()
	if err := logger.Configure(envs); err != nil {
		t.Fatal(err)
	}
	if err := logger.Open(ctx); err != nil {
		t.Fatal(err)
	}
	defer logger.Close(ctx)

	//setup metrics
	metrics := metrics.NewOpenTelemetry(logger)
	if err := metrics.Configure(envs); err != nil {
		t.Fatal(err)
	}
	if err := metrics.Open(ctx); err != nil {
		t.Fatal(err)
	}
	defer metrics.Close(ctx)

	//create metric and counter
	meterName := internal.GenerateId()
	t.Logf("Meter Name: %s", meterName)
	meter := metrics.Meter(meterName)
	counter, _ := meter.Int64Counter("request_count") //reuse the metric

	counter.Add(ctx, 1)
	counter.Add(ctx, 1)

	//wait for metrics to be published
	<-time.After(5 * time.Second)
}
