package client

import (
	"context"

	"github.com/antonio-alexander/go-blog-observability/internal/data"
)

type Client interface {
	EmployeeCreate(ctx context.Context, token string,
		employeePartial data.EmployeePartial) (*data.Employee, error)
	EmployeeRead(ctx context.Context, token string, empNo int64) (*data.Employee, error)
	EmployeesSearch(ctx context.Context, token string,
		search data.EmployeeSearch) ([]*data.Employee, error)
	EmployeeUpdate(ctx context.Context, token string, empNo int64,
		employeePartial data.EmployeePartial) (*data.Employee, error)
	EmployeeDelete(ctx context.Context, token string, empNo int64) error

	Sleep(ctx context.Context, sleep data.Sleep) (*data.Sleep, error)
	Panic(ctx context.Context) error
	CacheClear(ctx context.Context) error
}
