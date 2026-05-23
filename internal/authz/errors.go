package authz

import "github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"

const (
	errUnauthorized string = "not authorized"
	errJwt          string = "error parsing token"
)

var ErrJwt = errors.ErrorCommon{
	ErrorMessage: errJwt,
	ErrorType:    ErrorTypeUnauthorized,
	Local:        true,
}

func ErrUnauthorized(userId, action, dataType string, dataId *string) error {
	return ErrorUnauthorized{
		ErrorCommon: errors.ErrorCommon{
			ErrorMessage: errUnauthorized,
			ErrorType:    ErrorTypeUnauthorized,
			DataId:       dataId,
			DataType:     &dataType,
			Local:        true,
		},
		UserId: userId,
		Action: action,
	}
}
