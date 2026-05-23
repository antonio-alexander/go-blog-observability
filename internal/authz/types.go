package authz

import (
	"context"

	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
)

const packageName string = "github.com/antonio-alexander/go-blog-observability/internal/authz"

const (
	errMutationDisabled = "mutation disabled"
	errMeterNil         = "meter is nil"
)

var (
	ErrMutationDisabled = error(errors.ErrorCommon{
		ErrorMessage: errMutationDisabled,
		ErrorType:    errors.ErrorTypeUnknown,
	})
	ErrMeterNil = error(errors.ErrorCommon{
		ErrorMessage: errMeterNil,
		ErrorType:    errors.ErrorTypeUnknown,
	})
)

type Authz interface {
	EmployeeCreate(ctx context.Context, authorization string, employeePartial data.EmployeePartial) (*data.Employee, error)
	EmployeeRead(ctx context.Context, authorization string, empNo int64) (*data.Employee, error)
	EmployeesSearch(ctx context.Context, authorization string, search data.EmployeeSearch) ([]*data.Employee, error)
	EmployeeUpdate(ctx context.Context, authorization string, empNo int64, employeePartial data.EmployeePartial) (*data.Employee, error)
	EmployeeDelete(ctx context.Context, authorization string, empNo int64) error

	Sleep(ctx context.Context, sleepPartial data.Sleep) (*data.Sleep, error)
	Panic(ctx context.Context)
}
