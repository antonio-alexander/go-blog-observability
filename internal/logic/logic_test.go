package logic_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/cache"
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/logic"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"
	"github.com/antonio-alexander/go-blog-observability/internal/sql"

	"github.com/stretchr/testify/assert"
)

var (
	envs = map[string]string{
		//logger
		"LOG_LEVEL": "trace",
		//sql
		"DATABASE_HOST":          "localhost",
		"DATABASE_PORT":          "3306",
		"DATABASE_NAME":          "employees",
		"DATABASE_USER":          "mysql",
		"DATABASE_PASSWORD":      "mysql",
		"DATABASE_QUERY_TIMEOUT": "10",
		"DATABASE_PARSE_TIME":    "true",
		//cache
		"REDIS_ADDRESS":                "localhost",
		"REDIS_PORT":                   "6379",
		"REDIS_PASSWORD":               "",
		"REDIS_DATABASE":               "",
		"REDIS_TIMEOUT":                "10",
		"CACHE_PRUNE_INTERVAL":         "30",
		"CACHE_SET_READ_TTL":           "10",
		"CACHE_ENABLE_IN_PROGRESS":     "true",
		"CACHE_REDIS_MUTEX_EXPIRATION": "10",
		"REDIS_MUTEX_RETRY_INTERVAL":   "1",
		//logic
		"LOGIC_CACHE_ENABLED":     "true",
		"MUTATE_DISABLED":         "false",
		"CACHE_RETRY_INTERVAL":    "1",
		"CACHE_MAX_RETRIES":       "2",
		"CACHE_RETRY_EXP_BACKOFF": "true",
		//metrics
	}
)

func init() {
	for _, env := range os.Environ() {
		if key, value, ok := strings.Cut(env, "="); ok && value != "" {
			envs[key] = value
		}
	}
}

type logicTest struct {
	sql interface {
		internal.Configurer
		internal.Opener
		sql.Sql
	}
	cache interface {
		internal.Configurer
		internal.Opener
		cache.Cache
	}
	logic, logger,
	metrics, opentel interface {
		internal.Configurer
		internal.Opener
	}
	logic.Logic
}

func newLogicTest() *logicTest {
	logger := logger.NewSlog()
	metrics := metrics.NewOpenTelemetry(logger)
	sql := sql.New(logger)
	cache := cache.NewRedis(logger)
	logic := logic.NewLogic(sql, cache, logger, metrics)
	return &logicTest{
		logger:  logger,
		sql:     sql,
		cache:   cache,
		logic:   logic,
		Logic:   logic,
		metrics: metrics,
	}
}

func (l *logicTest) Configure(envs map[string]string) error {
	if err := l.logger.Configure(envs); err != nil {
		return err
	}
	if err := l.metrics.Configure(envs); err != nil {
		return err
	}
	if err := l.sql.Configure(envs); err != nil {
		return err
	}
	if err := l.cache.Configure(envs); err != nil {
		return err
	}
	if err := l.logic.Configure(envs); err != nil {
		return err
	}
	return nil
}

func (l *logicTest) Open(ctx context.Context) error {
	if err := l.logger.Open(ctx); err != nil {
		return err
	}
	if err := l.metrics.Open(ctx); err != nil {
		return err
	}
	if err := l.sql.Open(ctx); err != nil {
		return err
	}
	if err := l.cache.Open(ctx); err != nil {
		return err
	}
	if err := l.logic.Open(ctx); err != nil {
		return err
	}
	return nil
}

func (l *logicTest) Close(ctx context.Context) {
	l.logic.Close(ctx)
	l.cache.Close(ctx)
	l.sql.Close(ctx)
	l.metrics.Close(ctx)
	l.logger.Close(ctx)
}

func (l *logicTest) TestLogic(cacheEnabled bool) func(t *testing.T) {
	return func(t *testing.T) {
		// generate context
		ctx := context.TODO()

		// create employee
		birthDate, hireDate := time.Now().Unix(), time.Now().Unix()
		firstName := internal.GenerateId()[:14]
		lastName := internal.GenerateId()[:16]
		gender := "M"
		employeeCreated, err := l.EmployeeCreate(ctx, data.EmployeePartial{
			BirthDate: &birthDate,
			FirstName: &firstName,
			LastName:  &lastName,
			HireDate:  &hireDate,
			Gender:    &gender,
		})
		assert.Nil(t, err)
		assert.NotNil(t, employeeCreated)
		empNo := employeeCreated.EmpNo
		defer func(empNo int64) {
			_ = l.EmployeeDelete(ctx, empNo)
		}(empNo)

		if cacheEnabled {
			// validate that employee not in cache
			employeeCached, err := l.cache.EmployeeRead(ctx, empNo)
			assert.NotNil(t, err)
			assert.Nil(t, employeeCached)
		}

		// read employee
		employeeRead, err := l.EmployeeRead(ctx, empNo)
		assert.Nil(t, err)
		assert.NotNil(t, employeeRead)
		assert.Equal(t, employeeCreated, employeeRead)

		// validate that employee in cache
		if cacheEnabled {
			employeeCached, err := l.cache.EmployeeRead(ctx, empNo)
			assert.Nil(t, err)
			assert.NotNil(t, employeeCached)
			assert.Equal(t, employeeCreated, employeeCached)
		}

		// update employee
		updatedFirstName := internal.GenerateId()[:14]
		updatedLastName := internal.GenerateId()[:16]
		employeeUpdated, err := l.EmployeeUpdate(ctx, empNo, data.EmployeePartial{
			FirstName: &updatedFirstName,
			LastName:  &updatedLastName,
		})
		assert.Nil(t, err)
		assert.NotNil(t, employeeUpdated)

		// validate that employee not in cache
		if cacheEnabled {
			employeeCached, err := l.cache.EmployeeRead(ctx, empNo)
			assert.NotNil(t, err)
			assert.Nil(t, employeeCached)
		}

		// read employee
		employeeRead, err = l.EmployeeRead(ctx, empNo)
		assert.Nil(t, err)
		assert.NotNil(t, employeeRead)
		assert.Equal(t, employeeUpdated, employeeRead)

		// validate that employee in cache
		if cacheEnabled {
			employeeCached, err := l.cache.EmployeeRead(ctx, empNo)
			assert.Nil(t, err)
			assert.NotNil(t, employeeCached)
			assert.Equal(t, employeeUpdated, employeeCached)
		}

		// delete employee
		err = l.EmployeeDelete(ctx, empNo)
		assert.Nil(t, err)

		if cacheEnabled {
			// validate that employee not in cache
			employeeCached, err := l.cache.EmployeeRead(ctx, empNo)
			assert.NotNil(t, err)
			assert.Nil(t, employeeCached)
		}
	}
}

// determine hit/miss ratio with concurrent reads when
// invalidating the cache, possibly overall benchmark too
func (l *logicTest) TestLogicConcurrent(cacheEnabled bool) func(t *testing.T) {
	return func(t *testing.T) {
		const readInterval time.Duration = time.Second
		const updateInterval time.Duration = time.Second
		const nGoRoutines int = 5

		if !cacheEnabled {
			t.Skip("cache disabled")
		}

		var wg sync.WaitGroup

		// generate dynamic constants
		ctx := context.TODO()

		// create employee
		birthDate := time.Now().Unix()
		firstName := internal.GenerateId()[:14]
		lastName := internal.GenerateId()[:16]
		gender, hireDate := "M", time.Now().Unix()
		employeeCreated, err := l.EmployeeCreate(ctx, data.EmployeePartial{
			BirthDate: &birthDate,
			FirstName: &firstName,
			LastName:  &lastName,
			Gender:    &gender,
			HireDate:  &hireDate,
		})
		assert.Nil(t, err)
		assert.NotNil(t, employeeCreated)
		empNo := employeeCreated.EmpNo
		defer func(empNo int64) {
			_ = l.EmployeeDelete(ctx, empNo)
		}(empNo)

		//start read go routines
		start, stop := make(chan struct{}), make(chan struct{})
		for i := range nGoRoutines {
			wg.Add(1)
			go func(goRoutine int) {
				defer wg.Done()

				readEmployeeFx := func() {
					if _, err := l.EmployeeRead(ctx, empNo); err != nil {
						return
					}
				}
				tRead := time.NewTicker(readInterval)
				defer tRead.Stop()
				<-start
				for {
					select {
					case <-stop:
						return
					case <-tRead.C:
						readEmployeeFx()
					}
				}
			}(i)
		}

		//create go routine to write and delete policy data
		wg.Add(1)
		go func() {
			defer wg.Done()

			firstName := internal.GenerateId()[:14]
			lastName := internal.GenerateId()[:16]
			updateEmployeeFx := func() error {
				if _, err := l.EmployeeUpdate(ctx, empNo,
					data.EmployeePartial{
						FirstName: &firstName,
						LastName:  &lastName,
					}); err != nil {
					return err
				}
				return nil
			}
			tUpdate := time.NewTicker(updateInterval)
			defer tUpdate.Stop()
			<-start
			for {
				select {
				case <-stop:
					return
				case <-tUpdate.C:
					err := updateEmployeeFx()
					assert.Nil(t, err)
				}
			}
		}()

		//start the go routines
		close(start)

		//allow go routines to run
		<-time.After(10 * time.Second)

		//stop go routines
		close(stop)
		wg.Wait()
	}
}

func testLogic(t *testing.T) {
	c := newLogicTest()

	ctx := context.TODO()
	cacheEnabled, _ := strconv.ParseBool(envs["LOGIC_CACHE_ENABLED"])
	err := c.Configure(envs)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to configure testLogic")
	}
	err = c.Open(ctx)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to open testLogic")
	}
	defer c.Close(ctx)
	t.Run("Logic", c.TestLogic(cacheEnabled))
	t.Run("Concurrent Mutate", c.TestLogicConcurrent(cacheEnabled))
}

func TestLogic(t *testing.T) {
	testLogic(t)
}
