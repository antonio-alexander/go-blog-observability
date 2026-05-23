package cache

import (
	"context"
	"fmt"

	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
)

const (
	errEmployeeNotCached            = "employee not cached"
	errEmployeeNotFoundCached       = "employee not found; cached"
	errEmployeeReadSet              = "employee not cached, read set"
	errEmployeeReadAlreadySet       = "employee not cached, read already set"
	errEmployeeSearchNotCached      = "employee search not cached"
	errEmployeeSearchNotFoundCached = "employee search not found; cached"
	errEmployeesSearchSet           = "employees search not cached, read set"
	errEmployeesSearchAlreadySet    = "employees search not cached, read already set"
	errSleepNotCached               = "sleep not cached"
	errSleepNotFoundCached          = "sleep not found; cached"
	errSleepReadSet                 = "sleep not cached, read set"
	errSleepReadAlreadySet          = "sleep not cached, read already set"
)

var (
	ErrEmployeeSearchNotCached = error(errors.ErrorCommon{
		ErrorMessage: errEmployeeSearchNotCached,
		ErrorType:    errors.ErrorTypeNotCached,
		DataType:     new("employee"),
	})
	ErrEmployeeSearchNotFoundCached = error(errors.ErrorCommon{
		ErrorMessage: errEmployeeSearchNotFoundCached,
		ErrorType:    errors.ErrorTypeNotFound,
		DataType:     new("employee"),
	})
	ErrEmployeesSearchSet = error(errors.ErrorCommon{
		ErrorMessage: errEmployeesSearchSet,
		ErrorType:    errors.ErrorTypeNotCachedRetry,
		DataType:     new("employee"),
	})
	ErrEmployeesSearchAlreadySet = error(
		errors.ErrorCommon{
			ErrorMessage: errEmployeesSearchAlreadySet,
			ErrorType:    errors.ErrorTypeNotCachedRetry,
			DataType:     new("employee"),
		})
)

type Cache interface {
	EmployeeRead(ctx context.Context, empNo int64) (*data.Employee, error)
	EmployeesRead(ctx context.Context, search data.EmployeeSearch) ([]*data.Employee, error)
	EmployeesWrite(ctx context.Context, search data.EmployeeSearch, employees ...*data.Employee) error
	EmployeesDelete(ctx context.Context, empNos ...int64) error
	EmployeesNotFoundWrite(ctx context.Context, search data.EmployeeSearch, empNos ...int64) error

	SleepRead(ctx context.Context, sleepId string) (*data.Sleep, error)
	SleepWrite(ctx context.Context, sleep *data.Sleep) error
	SleepsDelete(ctx context.Context, sleepIds ...string) error
}

func ErrEmployeeNotCached(empNo int64) error {
	return errors.ErrorCommon{
		ErrorMessage: errEmployeeNotCached,
		ErrorType:    errors.ErrorTypeNotCached,
		DataId:       new(fmt.Sprint(empNo)),
		DataType:     new("employee"),
	}
}

func ErrEmployeeNotFoundCached(empNo int64) error {
	return errors.ErrorCommon{
		ErrorMessage: errEmployeeNotFoundCached,
		ErrorType:    errors.ErrorTypeNotFound,
		DataId:       new(fmt.Sprint(empNo)),
		DataType:     new("employee"),
	}
}

func ErrEmployeeReadSet(empNo int64) error {
	return errors.ErrorCommon{
		ErrorMessage: errEmployeeReadSet,
		ErrorType:    errors.ErrorTypeNotCachedRetry,
		DataId:       new(fmt.Sprint(empNo)),
		DataType:     new("employee"),
	}
}

func ErrEmployeeReadAlreadySet(empNo int64) error {
	return errors.ErrorCommon{
		ErrorMessage: errEmployeeReadAlreadySet,
		ErrorType:    errors.ErrorTypeNotCachedRetry,
		DataId:       new(fmt.Sprint(empNo)),
		DataType:     new("employee"),
	}
}

func ErrSleepNotCached(sleepId string) error {
	return errors.ErrorCommon{
		ErrorMessage: errSleepNotCached,
		ErrorType:    errors.ErrorTypeNotCached,
		DataId:       new(sleepId),
		DataType:     new("sleep"),
	}
}

func ErrSleepNotFoundCached(sleepId string) error {
	return errors.ErrorCommon{
		ErrorMessage: errSleepNotFoundCached,
		ErrorType:    errors.ErrorTypeNotFound,
		DataId:       new(sleepId),
		DataType:     new("sleep"),
	}
}

func ErrSleepReadSet(sleepId string) error {
	return errors.ErrorCommon{
		ErrorMessage: errSleepReadSet,
		ErrorType:    errors.ErrorTypeNotCachedRetry,
		DataId:       new(sleepId),
		DataType:     new("sleep"),
	}
}

func ErrSleepReadAlreadySet(sleepId string) error {
	return errors.ErrorCommon{
		ErrorMessage: errSleepReadAlreadySet,
		ErrorType:    errors.ErrorTypeNotCachedRetry,
		DataId:       new(sleepId),
		DataType:     new("sleep"),
	}
}
