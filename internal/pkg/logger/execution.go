package logger

import (
	"fmt"
	"log/slog"
)

func getAttributes(items []any) []slog.Attr {
	var attrs []slog.Attr

	for _, item := range items {
		switch v := item.(type) {
		case slog.Attr:
			attrs = append(attrs, v)
		case Loggable:
			attrs = append(attrs, v.GetAttributes()...)
		case error:
			attrs = append(attrs, slog.String("error", v.Error()))
		}
	}
	return attrs
}

func hasFormatIdentifiers(s string) bool {
	return fmt.Sprintf(s) != s
}
