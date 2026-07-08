package logic

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/cache"
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/tracer"
	"github.com/antonio-alexander/go-blog-observability/internal/sql"

	"github.com/cenkalti/backoff/v5"
)

type logic struct {
	logger.Logger
	tracer.Tracer
	metrics.Metrics
	config struct {
		cacheEnabled         bool
		cacheRetryInterval   int
		cacheMaxRetries      int
		cacheRetryExpBackoff bool
		cacheNotFoundEnabled bool
		mutateDisabled       bool
		metricsEnabled       bool
	}
	cache               cache.Cache
	sql                 sql.Sql
	backoffRetryOptions []backoff.RetryOption
}

func NewLogic(parameters ...any) interface {
	internal.Configurer
	internal.Opener
	Logic
} {
	l := &logic{}
	for _, parameter := range parameters {
		switch v := parameter.(type) {
		case sql.Sql:
			l.sql = v
		case cache.Cache:
			l.cache = v
		case metrics.Metrics:
			l.Metrics = v
		case tracer.Tracer:
			l.Tracer = v
		case logger.Logger:
			l.Logger = v
		}
	}
	return l
}

func (l *logic) Configure(envs map[string]string) error {
	//set default configuration
	l.config.cacheRetryInterval = 1
	l.config.cacheMaxRetries = 2
	l.config.cacheRetryExpBackoff = true

	//set configuration
	if cacheEnabled, ok := envs["LOGIC_CACHE_ENABLED"]; ok {
		l.config.cacheEnabled, _ = strconv.ParseBool(cacheEnabled)
	}
	if mutateDisabled, ok := envs["MUTATE_DISABLED"]; ok {
		l.config.mutateDisabled, _ = strconv.ParseBool(mutateDisabled)
	}
	if cacheRetryInterval, ok := envs["CACHE_RETRY_INTERVAL"]; ok {
		l.config.cacheRetryInterval, _ = strconv.Atoi(cacheRetryInterval)
	}
	if cacheMaxRetries, ok := envs["CACHE_MAX_RETRIES"]; ok {
		l.config.cacheMaxRetries, _ = strconv.Atoi(cacheMaxRetries)
	}
	if cacheRetryExpBackoff, ok := envs["CACHE_RETRY_EXP_BACKOFF"]; ok {
		l.config.cacheRetryExpBackoff, _ = strconv.ParseBool(cacheRetryExpBackoff)
	}
	if cacheNotFoundEnabled, ok := envs["CACHE_NOT_FOUND_ENABLED"]; ok {
		l.config.cacheNotFoundEnabled, _ = strconv.ParseBool(cacheNotFoundEnabled)
	}
	return nil
}

func (l *logic) Open(ctx context.Context) error {
	if l.config.cacheEnabled && l.cache == nil {
		return ErrCacheEnabledButNotSet
	}
	l.backoffRetryOptions = []backoff.RetryOption{
		backoff.WithMaxTries(uint(l.config.cacheMaxRetries)),
	}
	if l.config.cacheRetryExpBackoff {
		l.backoffRetryOptions = append(l.backoffRetryOptions,
			backoff.WithBackOff(backoff.NewExponentialBackOff()))
	}
	return nil
}

func (l *logic) Close(ctx context.Context) {
	//
}

func (l *logic) EmployeeCreate(ctx context.Context, employeePartial data.EmployeePartial) (*data.Employee, error) {
	ctx, span := l.Start(ctx, "logic.employee_create")
	defer span.End()
	if l.config.mutateDisabled {
		span.RecordError(ErrMutationDisabled)
		return nil, ErrMutationDisabled
	}
	employee, err := l.sql.EmployeeCreate(ctx, employeePartial)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return employee, nil
}

func (l *logic) EmployeeRead(ctx context.Context, empNo int64) (*data.Employee, error) {
	ctx, span := l.Start(ctx, "logic.EmployeeRead")
	defer span.End()
	if l.config.cacheEnabled {
		employee, err := backoff.Retry(ctx, func() (*data.Employee, error) {
			employee, err := l.cache.EmployeeRead(ctx, empNo)
			if err != nil {
				switch {
				default:
					return nil, backoff.Permanent(err)
				case errors.Is(err, errors.ErrTypeNotCachedInProgressSet):
					//if in progress set, we should read not attempt to
					// retry, but read directly from sql
					return nil, backoff.Permanent(err)
				case errors.Is(err, errors.ErrNotCached),
					errors.Is(err, errors.ErrNotCachedRetry):
					return nil, backoff.RetryAfter(l.config.cacheRetryInterval)
				case errors.Is(err, errors.ErrNotFound):
					return nil, backoff.Permanent(err)
				}
			}
			return employee, nil
		}, l.backoffRetryOptions...)
		if err == nil {
			return employee, nil
		}
		if l.config.cacheNotFoundEnabled &&
			(errors.Is(err, errors.ErrNotCached) ||
				errors.Is(err, errors.ErrNotCachedRetry)) {
			span.RecordError(err)
			return nil, err
		}
	}
	employee, err := l.sql.EmployeeRead(ctx, empNo)
	if err != nil {
		if l.config.cacheEnabled && errors.Is(err, errors.ErrNotFound) {
			if err := l.cache.EmployeesNotFoundWrite(ctx, data.EmployeeSearch{}, empNo); err != nil {
				l.Debug(ctx, "error while writing employee not found to cache",
					slog.Int64("emp_no", empNo), err)
			}
			span.RecordError(err)
			return nil, err
		}
		span.RecordError(err)
		return nil, err
	}
	if l.config.cacheEnabled {
		if err := l.cache.EmployeesWrite(ctx, data.EmployeeSearch{}, employee); err != nil {
			l.Debug(ctx, "error while writing employee to cache",
				slog.Int64("emp_no", empNo), err)
		}
	}
	return employee, nil
}

func (l *logic) EmployeesSearch(ctx context.Context, search data.EmployeeSearch) ([]*data.Employee, error) {
	var searchKey string
	var err error

	ctx, span := l.Start(ctx, "logic.EmployeesSearch")
	defer span.End()
	if l.config.cacheEnabled {
		searchKey, err = search.ToKey()
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		employees, err := backoff.Retry(ctx, func() ([]*data.Employee, error) {
			employees, err := l.cache.EmployeesRead(ctx, search)
			if err != nil {
				switch {
				default:
					return nil, backoff.Permanent(err)
				case errors.Is(err, errors.ErrNotCached),
					errors.Is(err, errors.ErrNotCachedRetry):
					return nil, backoff.RetryAfter(l.config.cacheRetryInterval)
				case errors.Is(err, errors.ErrNotFound):
					return nil, backoff.Permanent(err)
				}
			}
			return employees, nil
		}, l.backoffRetryOptions...)
		if err == nil {
			return employees, nil
		}
		if l.config.cacheNotFoundEnabled &&
			(errors.Is(err, errors.ErrNotCached) ||
				errors.Is(err, errors.ErrNotCachedRetry)) {
			span.RecordError(err)
			return nil, err
		}
	}
	employees, err := l.sql.EmployeesSearch(ctx, search)
	if err != nil {
		if l.config.cacheEnabled && errors.Is(err, errors.ErrNotFound) {
			if err := l.cache.EmployeesNotFoundWrite(ctx, search); err != nil {
				l.Debug(ctx, "error while writing employees not found to cache",
					slog.String("search_key", searchKey), err)
			}
		}
		span.RecordError(err)
		return nil, err
	}
	if l.config.cacheEnabled {
		if err := l.cache.EmployeesWrite(ctx, search, employees...); err != nil {
			l.Debug(ctx, "error while writing employees to cache: %s",
				slog.String("search_key", searchKey), err)
		}
	}
	return employees, nil
}

func (l *logic) EmployeeUpdate(ctx context.Context, empNo int64, employeePartial data.EmployeePartial) (*data.Employee, error) {
	ctx, span := l.Start(ctx, "logic.EmployeeUpdate")
	defer span.End()
	if l.config.mutateDisabled {
		span.RecordError(ErrMutationDisabled)
		return nil, ErrMutationDisabled
	}
	employee, err := l.sql.EmployeeUpdate(ctx, empNo, employeePartial)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	if l.config.cacheEnabled {
		if err := l.cache.EmployeesDelete(ctx, empNo); err != nil {
			l.Debug(ctx, "error while deleting employee from cache",
				slog.Int64("emp_no", empNo), err)
		} else {
			l.Debug(ctx, "cache invalidated", slog.Int64("emp_no", empNo))
		}
	}
	return employee, nil
}

func (l *logic) EmployeeDelete(ctx context.Context, empNo int64) error {
	ctx, span := l.Start(ctx, "logic.EmployeeDelete")
	defer span.End()
	if l.config.mutateDisabled {
		span.RecordError(ErrMutationDisabled)
		return ErrMutationDisabled
	}
	if err := l.sql.EmployeeDelete(ctx, empNo); err != nil {
		span.RecordError(err)
		return err
	}
	if l.config.cacheEnabled {
		if err := l.cache.EmployeesDelete(ctx, empNo); err != nil {
			l.Debug(ctx, "error while deleting employee from cache",
				slog.Int64("emp_no", empNo), err)
		}
	}
	return nil
}

func (l *logic) Sleep(ctx context.Context, s data.Sleep) (*data.Sleep, error) {
	ctx, span := l.Start(ctx, "logic.Sleep")
	defer span.End()
	if l.config.cacheEnabled {
		sleep, err := backoff.Retry(ctx, func() (*data.Sleep, error) {
			sleep, err := l.cache.SleepRead(ctx, s.Id)
			if err != nil {
				switch {
				default:
					return nil, backoff.Permanent(err)
				case errors.Is(err, errors.ErrNotCached),
					errors.Is(err, errors.ErrNotCachedRetry):
					return nil, backoff.RetryAfter(l.config.cacheRetryInterval)
				case errors.Is(err, errors.ErrNotFound):
					return nil, backoff.Permanent(err)
				}
			}
			return sleep, nil
		}, l.backoffRetryOptions...)
		if err == nil {
			return sleep, nil
		}
		if l.config.cacheNotFoundEnabled &&
			(errors.Is(err, errors.ErrNotCached) ||
				errors.Is(err, errors.ErrNotCachedRetry)) {
			return nil, err
		}
	}
	sleep, err := l.sql.Sleep(ctx, s)
	if err != nil {
		return nil, err
	}
	if l.config.cacheEnabled {
		if err := l.cache.SleepWrite(ctx, &s); err != nil {
			l.Debug(ctx, "error while writing sleep to cache",
				slog.String("sleep_id", s.Id), err)
		}
	}
	return sleep, nil
}

func (l *logic) Panic(ctx context.Context) {
	_, span := l.Start(ctx, "logic.Panic")
	defer span.End()
	panic(internal.GenerateId())
}
