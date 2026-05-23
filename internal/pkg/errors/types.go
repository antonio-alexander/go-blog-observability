package errors

var (
	ErrNotFound       error = &ErrorCommon{ErrorType: ErrorTypeNotFound}
	ErrNotCreated     error = &ErrorCommon{ErrorType: ErrorTypeNotCreated}
	ErrNotUpdated     error = &ErrorCommon{ErrorType: ErrorTypeNotUpdated}
	ErrConflict       error = &ErrorCommon{ErrorType: ErrorTypeConflict}
	ErrNotCached      error = &ErrorCommon{ErrorType: ErrorTypeNotCached}
	ErrNotCachedRetry error = &ErrorCommon{ErrorType: ErrorTypeNotCachedRetry}
	ErrValidation     error = &ErrorCommon{ErrorType: ErrorTypeValidation}
	ErrTimeout        error = &ErrorCommon{ErrorType: ErrorTypeTimeout}
	ErrUnknown        error = &ErrorCommon{ErrorType: ErrorTypeUnknown}
	ErrNotImplemented error = &ErrorCommon{ErrorType: ErrorTypeNotImplemented}
)
