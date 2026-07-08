package tracer

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
	TracerProvider() *sdktrace.TracerProvider
	PropagateTrace(ctx context.Context, r *http.Request)
	ExtractTrace(r *http.Request) context.Context
}
