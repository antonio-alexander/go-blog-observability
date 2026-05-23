package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/policy"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/tracer"
	"github.com/antonio-alexander/go-blog-observability/internal/service"
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
		//service
		"SERVICE_ADDRESS":                "localhost",
		"SERVICE_PORT":                   "8080",
		"SERVICE_SHUTDOWN_TIMEOUT":       "30",
		"SERVICE_CORS_ALLOW_CREDENTIALS": "",
		"SERVICE_CORS_ALLOWED_ORIGINS":   "",
		"SERVICE_CORS_ALLOWED_METHODS":   "",
		"SERVICE_CORS_DISABLED":          "",
		"SERVICE_CORS_DEBUG":             "",
	}
)

func init() {
	for _, env := range os.Environ() {
		if key, value, ok := strings.Cut(env, "="); ok && value != "" {
			envs[key] = value
		}
	}
}

type serviceTest struct {
	sql interface {
		internal.Configurer
		internal.Opener
		sql.Sql
	}
	logger, cache,
	logic, metrics,
	service, tracer,
	authz, policy interface {
		internal.Configurer
		internal.Opener
	}
	client                       *http.Client
	address                      string
	authzPrivateSigningKey       any
	authzPrivateSigningKeyMethod jwt.SigningMethod
}

func newServiceTest() *serviceTest {
	tracer := tracer.NewOpenTelemetry()
	metrics := metrics.NewOpenTelemetry()
	logger := logger.NewOpenTelemetry()
	policy := policy.New(logger, tracer)
	sql := sql.New(logger)
	cache := cache.NewRedis(logger)
	logic := logic.NewLogic(sql, cache, logger, metrics, tracer)
	authz := authz.New(logger, logic, metrics, tracer, policy)
	service := service.New(authz, logger, metrics, tracer)
	return &serviceTest{
		logger:  logger,
		sql:     sql,
		cache:   cache,
		logic:   logic,
		policy:  policy,
		tracer:  tracer,
		metrics: metrics,
		service: service,
		authz:   authz,
		client:  &http.Client{},
	}
}

func (s *serviceTest) doRequest(ctx context.Context, uri, method, authorization string, input any, v ...any) (any, error) {
	var byts []byte
	var err error

	switch v := input.(type) {
	default:
		if byts, err = json.Marshal(input); err != nil {
			return nil, err
		}
	case url.Values:
		uri += "?" + v.Encode()
	}
	body := bytes.NewBuffer(byts)
	request, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, err
	}
	if correlationId := pkgcontext.CorrelationIdFrom(ctx); correlationId != "" {
		request.Header.Add("Correlation-Id", correlationId)
	}
	if authorization != "" {
		request.Header.Add("Authorization", authorization)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	switch response.StatusCode {
	default:
		byts, _ = io.ReadAll(response.Body)
		if len(byts) > 0 {
			return nil, errors.Must(errors.New(fmt.Errorf("%s: %s", response.Status, string(byts))))
		}
		return nil, errors.Must(errors.New(fmt.Errorf("%s", response.Status)))
	case http.StatusNoContent:
		return []byte{}, nil
	case http.StatusOK:
		bytes, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		if len(v) > 0 {
			return bytes, json.Unmarshal(bytes, v[0])
		}
		return bytes, nil
	}
}

func (s *serviceTest) getPrivateSigningKey(envs map[string]string) error {
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
	s.authzPrivateSigningKeyMethod = authzPrivateSigningMethod
	s.authzPrivateSigningKey = authzPrivateSigningKey
	return nil
}

func (s *serviceTest) Configure(envs map[string]string) error {
	if err := s.getPrivateSigningKey(envs); err != nil {
		return err
	}
	if err := s.tracer.Configure(envs); err != nil {
		return err
	}
	if err := s.metrics.Configure(envs); err != nil {
		return err
	}
	if err := s.logger.Configure(envs); err != nil {
		return err
	}
	if err := s.policy.Configure(envs); err != nil {
		return err
	}
	if err := s.sql.Configure(envs); err != nil {
		return err
	}
	if err := s.cache.Configure(envs); err != nil {
		return err
	}
	if err := s.logic.Configure(envs); err != nil {
		return err
	}
	if err := s.authz.Configure(envs); err != nil {
		return err
	}
	if err := s.service.Configure(envs); err != nil {
		return err
	}
	s.address = "http://" + envs["SERVICE_ADDRESS"]
	if port := envs["SERVICE_PORT"]; port != "" {
		s.address += ":" + port
	}
	return nil
}

func (s *serviceTest) Open(ctx context.Context) error {
	if err := s.logger.Open(ctx); err != nil {
		return err
	}
	if err := s.metrics.Open(ctx); err != nil {
		return err
	}
	if err := s.tracer.Open(ctx); err != nil {
		return err
	}
	if err := s.policy.Open(ctx); err != nil {
		return err
	}
	if err := s.sql.Open(ctx); err != nil {
		return err
	}
	if err := s.cache.Open(ctx); err != nil {
		return err
	}
	if err := s.logic.Open(ctx); err != nil {
		return err
	}
	if err := s.authz.Open(ctx); err != nil {
		return err
	}
	if err := s.service.Open(ctx); err != nil {
		return err
	}
	time.Sleep(5 * time.Second) //wait for policy data to compile
	return nil
}

func (s *serviceTest) Close(ctx context.Context) {
	s.service.Close(ctx)
	s.authz.Close(ctx)
	s.logic.Close(ctx)
	s.cache.Close(ctx)
	s.sql.Close(ctx)
	s.policy.Close(ctx)
	s.metrics.Close(ctx)
	s.tracer.Close(ctx)
	s.logger.Close(ctx)
}

func (s *serviceTest) TestService(t *testing.T) {
	var request data.Request
	var response data.Response

	// generate correlationId
	correlationId := internal.GenerateId()
	t.Logf("correlation id: %s", correlationId)

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
	token, err := jwt.NewWithClaims(s.authzPrivateSigningKeyMethod,
		claims).SignedString(s.authzPrivateSigningKey)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to create token")
	}
	assert.NotEmpty(t, token)

	// create employee
	birthDate, hireDate := time.Now().Unix(), time.Now().Unix()
	firstName := internal.GenerateId()[:14]
	lastName := internal.GenerateId()[:16]
	gender := "M"
	employeeCreated := &data.Employee{}
	request.EmployeePartial = data.EmployeePartial{
		BirthDate: &birthDate,
		FirstName: &firstName,
		LastName:  &lastName,
		HireDate:  &hireDate,
		Gender:    &gender,
	}
	response.Employee = employeeCreated
	uriEmployeeCreate := s.address + data.RouteEmployees
	_, err = s.doRequest(ctx, uriEmployeeCreate, http.MethodPut, token, &request, &response)
	assert.Nil(t, err)
	assert.NotZero(t, employeeCreated.EmpNo)
	empNo := employeeCreated.EmpNo
	defer func(empNo int64) {
		uri := fmt.Sprintf(s.address+data.RouteEmployeesEmpNof, empNo)
		_, _ = s.doRequest(ctx, uri, http.MethodDelete, token, nil)
	}(empNo)

	// read employee
	employeeRead := &data.Employee{}
	response.Employee = employeeRead
	uriEmployeeRead := fmt.Sprintf(s.address+data.RouteEmployeesEmpNof, empNo)
	_, err = s.doRequest(ctx, uriEmployeeRead, http.MethodGet, token, nil, &response)
	assert.Nil(t, err)
	assert.Equal(t, employeeCreated, employeeRead)

	// read employees
	search := data.EmployeeSearch{EmpNos: []int64{empNo}}
	uriEmployeesSearch := s.address + data.RouteEmployeesSearch
	_, err = s.doRequest(ctx, uriEmployeesSearch, http.MethodGet,
		token, search.ToParams(), &response)
	assert.Nil(t, err)
	assert.Equal(t, employeeCreated, employeeRead)

	// update employee
	updatedFirstName := internal.GenerateId()[:14]
	updatedLastName := internal.GenerateId()[:16]
	employeeUpdated := &data.Employee{}
	request.EmployeePartial = data.EmployeePartial{
		FirstName: &updatedFirstName,
		LastName:  &updatedLastName,
	}
	response.Employee = employeeUpdated
	uriEmployeeUpdate := fmt.Sprintf(s.address+data.RouteEmployeesEmpNof, empNo)
	_, err = s.doRequest(ctx, uriEmployeeUpdate, http.MethodPost, token, &request, &response)
	assert.Nil(t, err)

	// delete employee
	uriEmployeeDelete := fmt.Sprintf(s.address+data.RouteEmployeesEmpNof, empNo)
	_, err = s.doRequest(ctx, uriEmployeeDelete, http.MethodDelete, token, nil)
	assert.Nil(t, err)
}

func testService(t *testing.T) {
	c := newServiceTest()

	ctx := context.TODO()
	err := c.Configure(envs)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to configure testService")
	}
	err = c.Open(ctx)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to open testService")
	}
	defer c.Close(ctx)
	t.Run("Service", c.TestService)
}

func TestService(t *testing.T) {
	testService(t)
}
