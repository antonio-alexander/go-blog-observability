package policy

import "github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"

const (
	errRego string = "unable to evaluate rego"
	errJson string = "unable to unmarshal/marshal with json"
)

func ErrJson(err error) error {
	return errors.Wrap(err, errors.ErrorCommon{
		Err:          err,
		ErrorMessage: errJson,
		ErrorType:    errors.ErrorTypeUnknown,
		Local:        true,
	})
}

func ErrRego(err error, query, file string) error {
	return errors.Wrap(err, ErrorRego{
		ErrorCommon: errors.ErrorCommon{
			Err:          err,
			ErrorMessage: errRego,
			ErrorType:    ErrorTypeRego,
			Local:        true,
		},
		Query:    query,
		RegoFile: file,
	})
}
