package cache_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/cache"
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"

	"github.com/stretchr/testify/assert"
)

var envs = map[string]string{
	//logger
	"LOG_LEVEL":     "trace",
	"REDIS_ADDRESS": "localhost",
	"REDIS_PORT":    "6379",
	"REDIS_TIMEOUT": "10",
	//stash_memory
	"STASH_EVICTION_POLICY": "least_recently_used",
	"STASH_TIME_TO_LIVE":    "0",
	"STASH_MAX_SIZE":        "5242880", //5MB
	"STASH_DEBUG_ENABLED":   "false",
	"STASH_DEBUG_PREFIX":    "stash ",
	//stash_redis
	"STASH_EVICTION_RATE": "1", //1s
}

func init() {
	for _, env := range os.Environ() {
		if key, value, ok := strings.Cut(env, "="); ok && value != "" {
			envs[key] = value
		}
	}
}

type cacheTest struct {
	logger interface {
		internal.Opener
		internal.Configurer
	}
	cache interface {
		internal.Opener
		internal.Configurer
		internal.Clearer
	}
	cache.Cache
}

func newCacheTest() *cacheTest {
	logger := logger.NewSlog()
	employeeCache := cache.NewRedis(logger)
	return &cacheTest{
		logger: logger,
		Cache:  employeeCache,
		cache:  employeeCache,
	}
}

func (c *cacheTest) TestCache(t *testing.T) {
	var search data.EmployeeSearch

	//create employee
	employees := []*data.Employee{
		{
			EmpNo:     1,
			FirstName: internal.GenerateId(),
			LastName:  internal.GenerateId(),
		},
		{
			EmpNo:     2,
			FirstName: internal.GenerateId(),
			LastName:  internal.GenerateId(),
		},
		{
			EmpNo:     3,
			FirstName: internal.GenerateId(),
			LastName:  internal.GenerateId(),
		},
		{
			EmpNo:     4,
			FirstName: internal.GenerateId(),
			LastName:  internal.GenerateId(),
		},
		{
			EmpNo:     5,
			FirstName: internal.GenerateId(),
			LastName:  internal.GenerateId(),
		},
	}

	//create context
	ctx := context.TODO()

	//clear cache
	err := c.cache.Clear(ctx)
	assert.Nil(t, err)

	// write employees
	err = c.EmployeesWrite(ctx, search, employees...)
	assert.Nil(t, err)

	// read employee[0]
	employeeRead, err := c.EmployeeRead(ctx, employees[0].EmpNo)
	assert.Nil(t, err)
	assert.Equal(t, employees[0], employeeRead)

	// read employee[1]
	employeeRead, err = c.EmployeeRead(ctx, employees[1].EmpNo)
	assert.Nil(t, err)
	assert.Equal(t, employees[1], employeeRead)

	// read employees
	for _, employee := range employees {
		search.EmpNos = append(search.EmpNos, employee.EmpNo)
	}
	_, err = c.EmployeesRead(ctx, search)
	assert.NotNil(t, err)

	// write search
	err = c.EmployeesWrite(ctx, search, employees...)
	assert.Nil(t, err)

	// read employees
	employeesRead, err := c.EmployeesRead(ctx, search)
	assert.Nil(t, err)
	assert.Equal(t, len(employees), len(employeesRead))
	for _, employee := range employees {
		assert.Contains(t, employeesRead, employee)
	}

	// delete employee [1]
	err = c.EmployeesDelete(ctx, employees[1].EmpNo)
	assert.Nil(t, err)

	//  attempt to read employee [1]
	employeeRead, err = c.EmployeeRead(ctx, employees[1].EmpNo)
	assert.NotNil(t, err)
	assert.Nil(t, employeeRead)
}

func testCache(t *testing.T) {
	c := newCacheTest()

	ctx := context.TODO()
	err := c.logger.Configure(envs)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to configure logger")
	}
	err = c.cache.Configure(envs)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to configure cache")
	}
	err = c.logger.Open(ctx)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to open logger")
	}
	defer c.logger.Close(ctx)
	err = c.cache.Open(ctx)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to open cache")
	}
	defer c.cache.Close(ctx)
	t.Run("Cache", c.TestCache)
}

func TestCache(t *testing.T) {
	testCache(t)
}
