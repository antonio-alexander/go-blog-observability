package sql

import (
	"context"
	"fmt"

	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
)

const (
	errEmployeeNotFound       = "employee not found"
	errEmployeeSearchNotFound = "employee search not found"
)

var ErrEmployeeSearchNotFound = error(errors.ErrorCommon{
	ErrorMessage: errEmployeeSearchNotFound,
	ErrorType:    errors.ErrorTypeNotFound,
	DataType:     new("employee"),
})

func ErrEmployeeNotFound(err error, empNo int64) error {
	dataId := fmt.Sprint(empNo)
	dataType := "employee"
	return errors.Wrap(errors.ErrorCommon{
		ErrorMessage: errEmployeeNotFound,
		ErrorType:    errors.ErrorTypeNotFound,
		DataId:       &dataId,
		DataType:     &dataType,
	}, err)
}

type Sql interface {
	EmployeeCreate(ctx context.Context, employeePartial data.EmployeePartial) (*data.Employee, error)
	EmployeeRead(ctx context.Context, empNo int64) (*data.Employee, error)
	EmployeesSearch(ctx context.Context, search data.EmployeeSearch) ([]*data.Employee, error)
	EmployeeUpdate(ctx context.Context, empNo int64, employeePartial data.EmployeePartial) (*data.Employee, error)
	EmployeeDelete(ctx context.Context, empNo int64) error

	Sleep(ctx context.Context, sleep data.Sleep) (*data.Sleep, error)
}
