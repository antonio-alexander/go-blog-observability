package pkgcontext

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
)

func ToAttributeKeyValue(ctx context.Context) []attribute.KeyValue {
	a := AttributesFrom(ctx)
	attrs := make([]attribute.KeyValue, 0, len(a))
	for _, attr := range a {
		switch attr.Value.Kind() {
		case slog.KindBool:
			attrs = append(attrs,
				attribute.Bool(attr.Key, attr.Value.Bool()))
		case slog.KindString:
			attrs = append(attrs,
				attribute.String(attr.Key, attr.Value.String()))
		case slog.KindInt64:
			attrs = append(attrs,
				attribute.Int64(attr.Key, attr.Value.Int64()))
		}
	}
	return attrs
}
