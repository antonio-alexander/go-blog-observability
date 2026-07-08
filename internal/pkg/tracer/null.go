package tracer

import (
	"context"
	"net/http"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type tracerNull struct{}

func NewNull(parameters ...any) interface {
	internal.Configurer
	internal.Opener
	Tracer
} {
	return &tracerNull{}
}

func (m *tracerNull) Configure(envs map[string]string) error {
	return nil
}

func (m *tracerNull) Open(ctx context.Context) error {
	return nil
}

func (m *tracerNull) Close(ctx context.Context) {
}

func (m *tracerNull) Start(ctx context.Context, spanName string,
	opts ...trace.SpanStartOption) (context.Context, Span) {
	return ctx, &nullSpan{}
}

func (m *tracerNull) TracerProvider() *sdktrace.TracerProvider {
	return nil
}

func (m *tracerNull) PropagateTrace(ctx context.Context, r *http.Request) {}

func (m *tracerNull) ExtractTrace(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}

type nullSpan struct{}

func (n *nullSpan) End(options ...trace.SpanEndOption)                  {}
func (n *nullSpan) SetAttributes(kv ...attribute.KeyValue)              {}
func (n *nullSpan) RecordError(err error, options ...trace.EventOption) {}
func (n *nullSpan) SetStatus(code codes.Code, description string)       {}
