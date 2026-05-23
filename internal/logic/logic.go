package logic

import (
	"context"
	"log/slog"
	"strconv"
	"sync"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/cache"
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/tracer"
	"github.com/antonio-alexander/go-blog-observability/internal/sql"

	"github.com/cenkalti/backoff/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type logic struct {
	logger.Logger
	metrics.Metrics
	tracer.Tracer
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
	meter               metrics.Meter
	counters            struct {
		sync.RWMutex
		data map[string]metrics.Int64Counter
	}
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

func (l *logic) createCounter(counterName string) (metrics.Int64UpDownCounter, error) {
	l.counters.Lock()
	defer l.counters.Unlock()
	counter, err := createCounter(l.meter, counterName)
	if err != nil {
		return nil, err
	}
	l.counters.data[counterName] = counter
	return counter, nil
}

func (l *logic) readCounter(counterName string) metrics.Int64Counter {
	l.counters.RLock()
	defer l.counters.RUnlock()
	return l.counters.data[counterName]
}

func (l *logic) addEmployeeHit(ctx context.Context, item any) {
	switch v := item.(type) {
	case int64:
		l.readCounter(counterNameEmployeeCacheHit).Add(ctx, 1,
			metric.WithAttributes(attribute.KeyValue{
				Key:   "emp_no",
				Value: attribute.Int64Value(v),
			}))
	case string:
		l.readCounter(counterNameEmployeeCacheHit).Add(ctx, 1,
			metric.WithAttributes(attribute.KeyValue{
				Key:   "emp_search_key",
				Value: attribute.StringValue(v),
			}))
	}
}

func (l *logic) addEmployeeMiss(ctx context.Context, item any) {
	switch v := item.(type) {
	case int64:
		l.readCounter(counterNameEmployeeCacheMiss).Add(ctx, 1,
			metric.WithAttributes(attribute.KeyValue{
				Key:   "emp_no",
				Value: attribute.Int64Value(v),
			}))
	case string:
		l.readCounter(counterNameEmployeeCacheMiss).Add(ctx, 1,
			metric.WithAttributes(attribute.KeyValue{
				Key:   "emp_search_key",
				Value: attribute.StringValue(v),
			}))
	}
}

func (l *logic) addSleepHit(ctx context.Context, sleepId string) {
	l.readCounter(counterNameSleepCacheHit).Add(ctx, 1,
		metric.WithAttributes(attribute.KeyValue{
			Key:   "sleep_id",
			Value: attribute.StringValue(sleepId),
		}))
}

func (l *logic) addSleepMiss(ctx context.Context, sleepId string) {
	l.readCounter(counterNameSleepCacheMiss).Add(ctx, 1,
		metric.WithAttributes(attribute.KeyValue{
			Key:   "sleep_id",
			Value: attribute.StringValue(sleepId),
		}))
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
	meter := l.Meter(packageName)
	l.meter = meter
	l.counters.data = make(map[string]metrics.Int64Counter)
	for _, counterName := range counterNames {
		if _, err := l.createCounter(counterName); err != nil {
			return err
		}
	}
	return nil
}

func (l *logic) Close(ctx context.Context) {
	l.meter = nil
	l.counters.Lock()
	defer l.counters.Unlock()
	l.counters.data = nil
}

func (l *logic) EmployeeCreate(ctx context.Context, employeePartial data.EmployeePartial) (*data.Employee, error) {
	ctx, span := l.Start(ctx, "logic.EmployeeCreate")
	defer span.End()
	if l.config.mutateDisabled {
		return nil, ErrMutationDisabled
	}
	return l.sql.EmployeeCreate(ctx, employeePartial)
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
				case errors.Is(err, errors.ErrNotCached),
					errors.Is(err, errors.ErrNotCachedRetry):
					l.Debug(ctx, "logic cache miss (retry) for employee",
						slog.Int64("emp_no", empNo), err)
					l.addEmployeeMiss(ctx, empNo)
					return nil, backoff.RetryAfter(l.config.cacheRetryInterval)
				case errors.Is(err, errors.ErrNotFound):
					return nil, backoff.Permanent(err)
				}
			}
			return employee, nil
		}, l.backoffRetryOptions...)
		if err == nil {
			l.Debug(ctx, "cache hit employee read cache hit", slog.Int64("emp_no", empNo))
			l.addEmployeeHit(ctx, empNo)
			return employee, nil
		}
		if l.config.cacheNotFoundEnabled &&
			(errors.Is(err, errors.ErrNotCached) ||
				errors.Is(err, errors.ErrNotCachedRetry)) {
			l.Debug(ctx, "cache hit (not found) for employee", slog.Int64("emp_no", empNo))
			l.addEmployeeHit(ctx, empNo)
			return nil, err
		}
		l.Debug(ctx, "cache miss (not found) for employee", slog.Int64("emp_no", empNo))
		// l.addEmployeeMiss(ctx, empNo)
	}
	employee, err := l.sql.EmployeeRead(ctx, empNo)
	if err != nil {
		if l.config.cacheEnabled && errors.Is(err, errors.ErrNotFound) {
			if err := l.cache.EmployeesNotFoundWrite(ctx, data.EmployeeSearch{}, empNo); err != nil {
				l.Debug(ctx, "error while writing employee not found to cache",
					slog.Int64("emp_no", empNo), err)
			}
			return nil, err
		}
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
					l.Debug(ctx, "search cache miss (retry)", err)
					return nil, backoff.RetryAfter(l.config.cacheRetryInterval)
				case errors.Is(err, errors.ErrNotFound):
					l.Debug(ctx, "cache hit for employee search read cache hit (not found)",
						slog.String("search_key", searchKey))
					l.addEmployeeHit(ctx, searchKey)
					return nil, backoff.Permanent(err)
				}
			}
			return employees, nil
		}, l.backoffRetryOptions...)
		if err == nil {
			l.Debug(ctx, "cache hit for employee search", slog.String("search_key", searchKey))
			l.addEmployeeHit(ctx, searchKey)
			return employees, nil
		}
		if l.config.cacheNotFoundEnabled &&
			(errors.Is(err, errors.ErrNotCached) ||
				errors.Is(err, errors.ErrNotCachedRetry)) {
			l.Debug(ctx, "cache hit (not found) for employee search ", slog.String("search_key", searchKey))
			l.addEmployeeHit(ctx, searchKey)
			return nil, err
		}
		l.Debug(ctx, "cache miss (not found) for employee search", slog.String("search_key", searchKey))
		l.addEmployeeMiss(ctx, searchKey)
	}
	employees, err := l.sql.EmployeesSearch(ctx, search)
	if err != nil {
		if l.config.cacheEnabled && errors.Is(err, errors.ErrNotFound) {
			if err := l.cache.EmployeesNotFoundWrite(ctx, search); err != nil {
				l.Debug(ctx, "error while writing employees not found to cache",
					slog.String("search_key", searchKey), err)
			}
		}
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
		return nil, ErrMutationDisabled
	}
	employee, err := l.sql.EmployeeUpdate(ctx, empNo, employeePartial)
	if err != nil {
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
		return ErrMutationDisabled
	}
	if err := l.sql.EmployeeDelete(ctx, empNo); err != nil {
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
					l.Debug(ctx, "cache miss (retry) for sleep",
						slog.String("sleep_id", s.Id), err)
					l.addSleepMiss(ctx, s.Id)
					return nil, backoff.RetryAfter(l.config.cacheRetryInterval)
				case errors.Is(err, errors.ErrNotFound):
					return nil, backoff.Permanent(err)
				}
			}
			return sleep, nil
		}, l.backoffRetryOptions...)
		if err == nil {
			l.Debug(ctx, "cache hit sleep read cache hit", slog.String("sleep_id", s.Id))
			l.addSleepHit(ctx, s.Id)
			return sleep, nil
		}
		if l.config.cacheNotFoundEnabled &&
			(errors.Is(err, errors.ErrNotCached) ||
				errors.Is(err, errors.ErrNotCachedRetry)) {
			l.Debug(ctx, "cache hit (not found) for sleep", slog.String("sleep_id", s.Id))
			l.addSleepHit(ctx, s.Id)
			return nil, err
		}
		l.Debug(ctx, "cache miss (not found) for sleep", slog.String("sleep_id", s.Id))
		l.addSleepMiss(ctx, s.Id)
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
