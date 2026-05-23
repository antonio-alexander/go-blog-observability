package policy

import (
	"fmt"
	"log/slog"

	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
)

const ErrorTypeRego errors.ErrorType = "ERR_REGO"

// ErrorRego represents an unauthorized error that can be generated
// swagger:model ErrorRego
type ErrorRego struct {
	errors.ErrorCommon

	Query    string `json:"query"`
	RegoFile string `json:"rego_file"`
}

func NewUnauthorized(item any) (*ErrorRego, error) {
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
	return &ErrorRego{
		ErrorCommon: errors.ErrorCommon{
			Err:          err,
			ErrorMessage: err.Error(),
			ErrorType:    ErrorTypeRego,
		},
	}, nil
}

func (e ErrorRego) GetAttributes() []slog.Attr {
	attrs := e.ErrorCommon.GetAttributes()
	attrs = append(attrs, slog.String("query", e.Query))
	attrs = append(attrs, slog.String("rego_file", e.RegoFile))
	return []slog.Attr{slog.GroupAttrs("error", attrs...)}
}
