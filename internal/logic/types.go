package logic

import (
	"context"

	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
)

const packageName string = "github.com/antonio-alexander/go-blog-observability/internal/logic"

const (
	errMutationDisabled      string = "mutation disabled"
	errCacheEnabledButNotSet string = "cache enabled, but no cache set/configured"
	errMetricsNil            string = "metrics are nil"
	errLoggerNil             string = "logger is nil"
	errMeterNil              string = "meter is nil"
)

var (
	ErrMutationDisabled = error(errors.ErrorCommon{
		ErrorMessage: errMutationDisabled,
		ErrorType:    errors.ErrorTypeUnknown,
	})
	ErrCacheEnabledButNotSet = error(errors.ErrorCommon{
		ErrorMessage: errCacheEnabledButNotSet,
		ErrorType:    errors.ErrorTypeUnknown,
	})
	ErrMetricsNil = error(errors.ErrorCommon{
		ErrorMessage: errMetricsNil,
		ErrorType:    errors.ErrorTypeUnknown,
	})
	ErrLoggerNil = error(errors.ErrorCommon{
		ErrorMessage: errLoggerNil,
		ErrorType:    errors.ErrorTypeUnknown,
	})
	ErrMeterNil = error(errors.ErrorCommon{
		ErrorMessage: errMeterNil,
		ErrorType:    errors.ErrorTypeUnknown,
	})
)

type Logic interface {
	EmployeeCreate(ctx context.Context, employeePartial data.EmployeePartial) (*data.Employee, error)
	EmployeeRead(ctx context.Context, empNo int64) (*data.Employee, error)
	EmployeesSearch(ctx context.Context, search data.EmployeeSearch) ([]*data.Employee, error)
	EmployeeUpdate(ctx context.Context, empNo int64, employeePartial data.EmployeePartial) (*data.Employee, error)
	EmployeeDelete(ctx context.Context, empNo int64) error

	Sleep(ctx context.Context, sleepPartial data.Sleep) (*data.Sleep, error)

	Panic(ctx context.Context)
}
