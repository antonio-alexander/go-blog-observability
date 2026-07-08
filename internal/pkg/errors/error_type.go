package errors

type ErrorType string

const (
	ErrorTypeNotFound               ErrorType = "ERR_NOT_FOUND"
	ErrorTypeNotCreated             ErrorType = "ERR_NOT_CREATED"
	ErrorTypeNotUpdated             ErrorType = "ERR_NOT_UPDATED"
	ErrorTypeConflict               ErrorType = "ERR_CONFLICT"
	ErrorTypeNotCached              ErrorType = "ERR_NOT_CACHED"
	ErrorTypeNotCachedRetry         ErrorType = "ERR_NOT_CACHED_RETRY"
	ErrorTypeNotCachedInProgressSet ErrorType = "ERR_NOT_CACHED_IN_PROGRESS_SET"
	ErrorTypeValidation             ErrorType = "ERR_VALIDATION"
	ErrorTypeTimeout                ErrorType = "ERR_TIMEOUT"
	ErrorTypeUnknown                ErrorType = "ERR_UNKNOWN"
	ErrorTypeNotImplemented         ErrorType = "ERR_NOT_IMPLEMENTED"
	ErrorTypeExciting               ErrorType = "ERR_EXCITING"
	ErrorTypeTooMuchData            ErrorType = "ERR_TOO_MUCH_DATA"
)

func (e ErrorType) String() string {
	return string(e)
}
