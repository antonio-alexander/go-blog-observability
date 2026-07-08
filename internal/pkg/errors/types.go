package errors

var (
	ErrNotFound                   = &ErrorCommon{ErrorType: ErrorTypeNotFound}
	ErrNotCreated                 = &ErrorCommon{ErrorType: ErrorTypeNotCreated}
	ErrNotUpdated                 = &ErrorCommon{ErrorType: ErrorTypeNotUpdated}
	ErrConflict                   = &ErrorCommon{ErrorType: ErrorTypeConflict}
	ErrNotCached                  = &ErrorCommon{ErrorType: ErrorTypeNotCached}
	ErrTypeNotCachedInProgressSet = &ErrorCommon{ErrorType: ErrorTypeNotCachedInProgressSet}
	ErrNotCachedRetry             = &ErrorCommon{ErrorType: ErrorTypeNotCachedRetry}
	ErrValidation                 = &ErrorCommon{ErrorType: ErrorTypeValidation}
	ErrTimeout                    = &ErrorCommon{ErrorType: ErrorTypeTimeout}
	ErrUnknown                    = &ErrorCommon{ErrorType: ErrorTypeUnknown}
	ErrNotImplemented             = &ErrorCommon{ErrorType: ErrorTypeNotImplemented}
)
