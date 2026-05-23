package errors

import (
	"errors"
	"fmt"
	"log/slog"
)

// ErrorCommon represents a common error that can be generated
// swagger:model ErrorCommon
type ErrorCommon struct {
	//Err is a nested value that contains all wrapped
	// errors in its native format
	Err error `json:"-"`

	// ErrorMessage contains the simple (static) text
	// of the error
	// example: employee not found
	ErrorMessage string `json:"error_message"`

	// ErrorMessageDetail contains the complex (dynamic) text
	// of the error (or wrapped errors)
	ErrorMessageDetail string `json:"error_message_detail"`

	// Type contains the text of the type of error
	// example: not found
	ErrorType ErrorType `json:"error_type"`

	// DataId contains the relevant id if there was
	// one involved in the error itself
	DataId *string `json:"id,omitempty"`

	// DataType contains the relevant data type
	// if there was one involved in the error itself
	DataType *string `json:"data_type,omitempty"`

	//Local can be used to determine if an error
	// occurred in the specific instance
	Local bool `json:"-"`
}

func New(item any, e ...ErrorType) (*ErrorCommon, error) {
	var err error

	errorType := ErrorTypeUnknown
	if len(e) > 0 {
		errorType = e[0]
	}
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
		err = errors.New(v)
	}
	return &ErrorCommon{
		Err:          err,
		ErrorMessage: err.Error(),
		ErrorType:    errorType,
	}, nil
}

func (e ErrorCommon) Error() string {
	if e.Err != nil {
		return e.ErrorMessage + ":" + e.Err.Error()
	}
	return e.ErrorMessage
}

func (e ErrorCommon) Type() string {
	return string(e.ErrorType)
}

func (e ErrorCommon) Is(err error) bool {
	if err, ok := err.(interface {
		Type() string
	}); ok {
		return string(e.ErrorType) == err.Type()
	}
	return false
}

func (e ErrorCommon) GetAttributes() []slog.Attr {
	attrs := make([]slog.Attr, 0, 4)
	attrs = append(attrs, slog.String("type", e.ErrorType.String()))
	attrs = append(attrs, slog.String("message", e.ErrorMessage))
	attrs = append(attrs, slog.String("message_detail", e.ErrorMessageDetail))
	attrs = append(attrs, slog.Bool("local", e.Local))
	if e.DataType != nil {
		attrs = append(attrs, slog.String("data_type", *e.DataType))
	}
	if e.DataId != nil {
		attrs = append(attrs, slog.String("data_id", *e.DataId))
	}
	return []slog.Attr{slog.GroupAttrs("error", attrs...)}
}
