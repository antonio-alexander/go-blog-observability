package authz

import (
	"fmt"
	"log/slog"

	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
)

const ErrorTypeUnauthorized errors.ErrorType = "ERR_NOT_AUTHORIZED"

// ErrorUnauthorized represents an unauthorized error that can be generated
// swagger:model ErrorUnauthorized
type ErrorUnauthorized struct {
	errors.ErrorCommon

	UserId string `json:"user_id"`
	Action string `json:"action"`
}

func NewUnauthorized(item any) (*ErrorUnauthorized, error) {
	var err error

	switch v := item.(type) {
	default:
		dataType := fmt.Sprintf("%T", item)
		return nil, &errors.ErrorCommon{
			ErrorMessage: "unsupported error type",
			ErrorType:    errors.ErrorTypeValidation,
			DataType:     &dataType,
		}
	case error:
		err = v
	case string:
		err = errors.Must(errors.New(v))
	}
	return &ErrorUnauthorized{
		ErrorCommon: errors.ErrorCommon{
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
