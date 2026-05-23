package tracer

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Span interface {
	End(options ...trace.SpanEndOption)
	SetAttributes(kv ...attribute.KeyValue)
}

type Tracer interface {
	Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, Span)
}
