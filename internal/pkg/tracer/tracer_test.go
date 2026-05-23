package tracer_test

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/tracer"

	"go.opentelemetry.io/otel/attribute"
)

var envs = map[string]string{
	"OTEL_LOG_LEVEL":      "info",
	"TRACE_ENABLE_STDOUT": "true",
}

func init() {
	envs := make(map[string]string)
	for _, env := range os.Environ() {
		if key, value, ok := strings.Cut(env, "="); ok && value != "" {
			envs[key] = value
		}
	}
}

func parentOperation(ctx context.Context, tracer tracer.Tracer) {
	// Start a parent span
	ctx, span := tracer.Start(ctx, "parent-work")
	defer span.End() // Always end the span!

	log.Println("Doing parent work...")
	childOperation(ctx, tracer)
}

func childOperation(ctx context.Context, tracer tracer.Tracer) {
	// Start a nested child span using the context from the parent
	_, span := tracer.Start(ctx, "child-work")
	defer span.End()

	// Add custom attributes
	span.SetAttributes(attribute.String("db.system", "postgres"))
	log.Println("Doing child work...")
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

	//setup tracer
	tracer := tracer.NewOpenTelemetry(logger)
	if err := tracer.Configure(envs); err != nil {
		t.Fatal(err)
	}
	if err := tracer.Open(ctx); err != nil {
		t.Fatal(err)
	}
	defer tracer.Close(ctx)

	//execute operation
	parentOperation(ctx, tracer)

	//wait for metrics to be published
	<-time.After(5 * time.Second)
}
