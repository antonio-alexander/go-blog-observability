package authz_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/authz"
	"github.com/antonio-alexander/go-blog-observability/internal/cache"
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/logic"
	pkgcontext "github.com/antonio-alexander/go-blog-observability/internal/pkg/context"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/policy"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/tracer"
	"github.com/antonio-alexander/go-blog-observability/internal/sql"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

var (
	userId = "14f3d416-6739-4b7f-94c0-74d50d6aa61f" //internal.GenerateId()
	envs   = map[string]string{
		//logger
		"LOG_LEVEL": "trace",
		// trace
		"TRACE_ENABLE_STDOUT": "false",
		//metrics
		"METRICS_ENABLE_STDOUT": "false",
		//rego/policy
		"REGO_COMPILE_FILENAME": "../../opa/compilation/",
		"EVAL_COMPILE_FILENAME": "../../opa/evaluation",
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
		//authz
		"AUTHZ_POLICY_COMPILE_INPUT_FILE": "../../data/policy_data.json",
		"AUTHZ_DISABLED":                  "false",
		"AUTHZ_PUBLIC_SIGNING_METHOD":     "HS256",
		"AUTHZ_PUBLIC_SIGNING_KEY":        "secret",
		"AUTHZ_PUBLIC_SIGNING_KEY_FILE":   "",
		"AUTHZ_PRIVATE_SIGNING_METHOD":    "HS256",
		"AUTHZ_PRIVATE_SIGNING_KEY":       "secret",
		"AUTHZ_PRIVATE_SIGNING_KEY_FILE":  "",
		"AUTHZ_USER_ID":                   userId,
		//redis
	}
)

func init() {
	for _, env := range os.Environ() {
		if key, value, ok := strings.Cut(env, "="); ok && value != "" {
			envs[key] = value
		}
	}
}

type authzTest struct {
	sql interface {
		internal.Configurer
		internal.Opener
	}
	cache, logic,
	logger, authz,
	tracer, metrics,
	policy interface {
		internal.Configurer
		internal.Opener
	}
	authzPrivateSigningKey       any
	authzPrivateSigningKeyMethod jwt.SigningMethod
	authz.Authz
}

func newLogicTest() *authzTest {
	tracer := tracer.NewOpenTelemetry()
	metrics := metrics.NewOpenTelemetry()
	logger := logger.NewSlog()
	policy := policy.New(logger, tracer)
	sql := sql.New(logger)
	cache := cache.NewRedis(logger)
	logic := logic.NewLogic(sql, cache, logger, metrics, tracer)
	authz := authz.New(logger, logic, metrics, tracer, policy)
	return &authzTest{
		logger:  logger,
		metrics: metrics,
		policy:  policy,
		sql:     sql,
		tracer:  tracer,
		cache:   cache,
		authz:   authz,
		logic:   logic,
		Authz:   authz,
	}
}

func (a *authzTest) getPrivateSigningKey(envs map[string]string) error {
	var privateSigningKeyMethod, privateSigningKey, privateSigningKeyFile string

	if s, ok := envs["AUTHZ_PRIVATE_SIGNING_KEY"]; ok {
		privateSigningKey = s
	}
	if s, ok := envs["AUTHZ_PRIVATE_SIGNING_METHOD"]; ok {
		privateSigningKeyMethod = s
	}
	if s, ok := envs["AUTHZ_PRIVATE_SIGNING_KEY_FILE"]; ok {
		privateSigningKeyFile = s
	}
	authzPrivateSigningMethod := jwt.GetSigningMethod(privateSigningKeyMethod)
	authzPrivateSigningKey, err := internal.GetPrivateSigningKey(privateSigningKeyMethod,
		privateSigningKey, privateSigningKeyFile)
	if err != nil {
		return err
	}
	a.authzPrivateSigningKeyMethod = authzPrivateSigningMethod
	a.authzPrivateSigningKey = authzPrivateSigningKey
	return nil
}

func (a *authzTest) Configure(envs map[string]string) error {
	if err := a.getPrivateSigningKey(envs); err != nil {
		return err
	}
	if err := a.tracer.Configure(envs); err != nil {
		return err
	}
	if err := a.metrics.Configure(envs); err != nil {
		return err
	}
	if err := a.logger.Configure(envs); err != nil {
		return err
	}
	if err := a.policy.Configure(envs); err != nil {
		return err
	}
	if err := a.sql.Configure(envs); err != nil {
		return err
	}
	if err := a.cache.Configure(envs); err != nil {
		return err
	}
	if err := a.logic.Configure(envs); err != nil {
		return err
	}
	if err := a.authz.Configure(envs); err != nil {
		return err
	}
	return nil
}

func (a *authzTest) Open(ctx context.Context) error {
	if err := a.logger.Open(ctx); err != nil {
		return err
	}
	if err := a.metrics.Open(ctx); err != nil {
		return err
	}
	if err := a.tracer.Open(ctx); err != nil {
		return err
	}
	if err := a.policy.Open(ctx); err != nil {
		return err
	}
	if err := a.sql.Open(ctx); err != nil {
		return err
	}
	if err := a.cache.Open(ctx); err != nil {
		return err
	}
	if err := a.logic.Open(ctx); err != nil {
		return err
	}
	if err := a.authz.Open(ctx); err != nil {
		return err
	}
	time.Sleep(5 * time.Second) //wait for policy data to compile
	return nil
}

func (a *authzTest) Close(ctx context.Context) {
	a.authz.Close(ctx)
	a.logic.Close(ctx)
	a.cache.Close(ctx)
	a.sql.Close(ctx)
	a.policy.Close(ctx)
	a.metrics.Close(ctx)
	a.tracer.Close(ctx)
	a.logger.Close(ctx)
}

func (a *authzTest) TestAuthz(t *testing.T) {
	const correlationId string = "test_authz"

	//generate context
	ctx := pkgcontext.WithCorrelationId(t.Context(), correlationId)

	// create token
	claims := &data.AuthzClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Minute)},
			ID:        internal.GenerateId(),
		},
		UserId: userId,
	}
	token, err := jwt.NewWithClaims(a.authzPrivateSigningKeyMethod,
		claims).SignedString(a.authzPrivateSigningKey)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to create token")
	}
	assert.NotEmpty(t, token)

	//create employee
	employeeCreated, err := a.EmployeeCreate(ctx, token, data.EmployeePartial{
		BirthDate: new(time.Now().Unix()),
		FirstName: new(internal.GenerateId()[:14]),
		LastName:  new(internal.GenerateId()[:16]),
		HireDate:  new(time.Now().Unix()),
		Gender:    new("M"),
	})
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to create employee")
	}
	assert.NotNil(t, employeeCreated)
	empNo := employeeCreated.EmpNo
	defer func(empNo int64) {
		_ = a.EmployeeDelete(ctx, token, empNo)
	}(empNo)

	//read employee
	employeeRead, err := a.EmployeeRead(ctx, token, empNo)
	assert.Nil(t, err)
	assert.NotNil(t, employeeRead)
	assert.Equal(t, employeeCreated, employeeRead)

	// read emloyees
	employeesRead, err := a.EmployeesSearch(ctx, token,
		data.EmployeeSearch{EmpNos: []int64{empNo}})
	assert.Nil(t, err)
	assert.Len(t, employeesRead, 1)
	assert.Contains(t, employeesRead, employeeCreated)

	//update employee
	employeeUpdated, err := a.EmployeeUpdate(ctx, token, empNo,
		data.EmployeePartial{
			FirstName: new(internal.GenerateId()[:14]),
		})
	assert.Nil(t, err)
	assert.NotNil(t, employeeUpdated)

	//delete employee
	err = a.EmployeeDelete(ctx, token, empNo)
	assert.Nil(t, err)
}

func testAuthz(t *testing.T) {
	ctx := t.Context()
	c := newLogicTest()

	err := c.Configure(envs)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to configure testAuthz")
	}
	err = c.Open(ctx)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to open testAuthz")
	}
	defer c.Close(ctx)
	t.Run("Authz", c.TestAuthz)
}

func TestAuthz(t *testing.T) {
	testAuthz(t)
}
