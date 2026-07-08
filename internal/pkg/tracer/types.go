package tracer

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Span interface {
	End(options ...trace.SpanEndOption)
	RecordError(err error, options ...trace.EventOption)
	SetAttributes(kv ...attribute.KeyValue)
	SetStatus(code codes.Code, description string)
}

type Tracer interface {
	Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, Span)
}
