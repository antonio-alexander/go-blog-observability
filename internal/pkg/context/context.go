package pkgcontext

import (
	"context"
	"log/slog"
)

type ctxKeyCorrelationId struct{}

func WithCorrelationId(ctx context.Context, correlationId string) context.Context {
	//store correlation id in both the log attributes and the correlation id
	// context value
	ctx = context.WithValue(ctx, ctxKeyCorrelationId{}, correlationId)
	return WithAttributes(ctx, slog.String("correlation_id", correlationId))
}

func CorrelationIdFrom(ctx context.Context) string {
	correlationId, ok := ctx.Value(ctxKeyCorrelationId{}).(string)
	if ok {
		return correlationId
	}
	return ""
}

type ctxKeyRequestId struct{}

func WithRequestId(ctx context.Context, requestId string) context.Context {
	//store request id in both the log attributes and the request id
	// context value
	ctx = context.WithValue(ctx, ctxKeyRequestId{}, requestId)
	return WithAttributes(ctx, slog.String("request_id", requestId))
}

func RequestIdFrom(ctx context.Context) string {
	requestId, ok := ctx.Value(ctxKeyRequestId{}).(string)
	if ok {
		return requestId
	}
	return ""
}

type ctxKeyTraceId struct{}

func WithTraceId(ctx context.Context, traceId string) context.Context {
	//store trace id in both the log attributes and the trace id
	// context value
	ctx = context.WithValue(ctx, ctxKeyTraceId{}, traceId)
	return WithAttributes(ctx, slog.String("request_id", traceId))
}

func TraceIdFrom(ctx context.Context) string {
	traceId, ok := ctx.Value(ctxKeyTraceId{}).(string)
	if ok {
		return traceId
	}
	return ""
}

type ctxKeyHostname struct{}

func WithHostname(ctx context.Context, hostname string) context.Context {
	//store trace id in both the log attributes and the trace id
	// context value
	ctx = context.WithValue(ctx, ctxKeyHostname{}, hostname)
	return WithAttributes(ctx, slog.String("host_name", hostname))
}

func HostnameFrom(ctx context.Context) string {
	hostname, ok := ctx.Value(ctxKeyHostname{}).(string)
	if ok {
		return hostname
	}
	return ""
}

type ctxKeyAttributes struct{}

func WithAttributes(ctx context.Context, attrs ...slog.Attr) context.Context {
	if existingAttrs := AttributesFrom(ctx); len(existingAttrs) > 0 {
		return context.WithValue(ctx, ctxKeyAttributes{},
			append(attrs, existingAttrs...))
	}
	return context.WithValue(ctx, ctxKeyAttributes{}, attrs)
}

func AttributesFrom(ctx context.Context) []slog.Attr {
	a, ok := ctx.Value(ctxKeyAttributes{}).([]slog.Attr)
	if ok {
		attrs := make([]slog.Attr, 0, len(a))
		for _, a := range a {
			if a.Value.String() == "" {
				continue
			}
			attrs = append(attrs, a)
		}
		return attrs
	}
	return nil
}
