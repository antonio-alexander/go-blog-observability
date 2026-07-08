package errors

import (
	"fmt"
	"log/slog"
)

const ErrorTypeUnauthorized ErrorType = "ERR_NOT_AUTHORIZED"

// ErrorUnauthorized represents an unauthorized error that can be generated
// swagger:model ErrorUnauthorized
type ErrorUnauthorized struct {
	ErrorCommon

	UserId string `json:"user_id"`
	Action string `json:"action"`
}

func NewUnauthorized(item any) (*ErrorUnauthorized, error) {
	var err error

	switch v := item.(type) {
	default:
		dataType := fmt.Sprintf("%T", item)
		return nil, &ErrorCommon{
			ErrorMessage: "unsupported error type",
			ErrorType:    ErrorTypeValidation,
			DataType:     &dataType,
		}
	case error:
		err = v
	case string:
		err = Must(New(v))
	}
	return &ErrorUnauthorized{
		ErrorCommon: ErrorCommon{
			Err:          err,
			ErrorMessage: err.Error(),
			ErrorType:    ErrorTypeUnauthorized,
		},
	}, nil
}

func (e ErrorUnauthorized) GetAttributes() []slog.Attr {
	attrs := e.ErrorCommon.GetAttributes()
	attrs = append(attrs, slog.String("user_id", e.UserId))
	attrs = append(attrs, slog.String("action", e.Action))
	return []slog.Attr{slog.GroupAttrs("error", attrs...)}
}
