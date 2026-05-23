package tracer

import (
	"context"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"go.opentelemetry.io/otel/attribute"
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

type nullSpan struct{}

func (n *nullSpan) End(options ...trace.SpanEndOption)     {}
func (n *nullSpan) SetAttributes(kv ...attribute.KeyValue) {}
