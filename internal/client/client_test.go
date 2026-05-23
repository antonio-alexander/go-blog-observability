package client_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/client"
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	pkgcontext "github.com/antonio-alexander/go-blog-observability/internal/pkg/context"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
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
		//authz
		"AUTHZ_PRIVATE_SIGNING_METHOD":   "HS256",
		"AUTHZ_PRIVATE_SIGNING_KEY":      "secret",
		"AUTHZ_PRIVATE_SIGNING_KEY_FILE": "",
		"AUTHZ_USER_ID":                  userId,
		//client
		"CLIENT_ADDRESS":  "localhost",
		"CLIENT_PORT":     "8081",
		"CLIENT_PROTOCOL": "http",
		"CLIENT_TIMEOUT":  "10",
		"SSL_CA_FILE":     "",
		"SSL_KEY_FILE":    "",
		"SSL_CRT_FILE":    "",
		"CACHE_DISABLED":  "false",
	}
)

func init() {
	for _, env := range os.Environ() {
		if key, value, ok := strings.Cut(env, "="); ok && value != "" {
			envs[key] = value
		}
	}
}

type clientTest struct {
	logger, client interface {
		internal.Opener
		internal.Configurer
	}
	client.Client
	authzPrivateSigningKey       any
	authzPrivateSigningKeyMethod jwt.SigningMethod
}

func newClientTest() *clientTest {
	logger := logger.NewSlog()
	client := client.NewClient(logger)
	return &clientTest{
		logger: logger,
		client: client,
		Client: client,
	}
}

func (c *clientTest) getPrivateSigningKey(envs map[string]string) error {
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
	c.authzPrivateSigningKeyMethod = authzPrivateSigningMethod
	c.authzPrivateSigningKey = authzPrivateSigningKey
	return nil
}

func (c *clientTest) Configure(envs map[string]string) error {
	if err := c.getPrivateSigningKey(envs); err != nil {
		return err
	}
	if err := c.logger.Configure(envs); err != nil {
		return err
	}
	if err := c.client.Configure(envs); err != nil {
		return err
	}
	return nil
}

func (c *clientTest) Open(ctx context.Context) error {
	if err := c.logger.Open(ctx); err != nil {
		return err
	}
	if err := c.client.Open(ctx); err != nil {
		return err
	}
	return nil
}

func (c *clientTest) Close(ctx context.Context) {
	c.client.Close(ctx)
	c.logger.Close(ctx)
}

func (c *clientTest) TestClient(t *testing.T) {
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
	token, err := jwt.NewWithClaims(c.authzPrivateSigningKeyMethod,
		claims).SignedString(c.authzPrivateSigningKey)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to create token")
	}
	assert.NotEmpty(t, token)

	// create employee
	birthDate, hireDate := time.Now().Unix(), time.Now().Unix()
	firstName := internal.GenerateId()[:14]
	lastName := internal.GenerateId()[:16]
	gender := "M"
	employeeCreated, err := c.EmployeeCreate(ctx, token,
		data.EmployeePartial{
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
		_ = c.EmployeeDelete(ctx, token, empNo)
	}(empNo)

	// read employee
	employeeRead, err := c.EmployeeRead(ctx, token, empNo)
	assert.Nil(t, err)
	assert.NotNil(t, employeeRead)
	assert.Equal(t, employeeCreated, employeeRead)

	// update employee
	updatedFirstName := internal.GenerateId()[:14]
	updatedLastName := internal.GenerateId()[:16]
	employeeUpdated, err := c.EmployeeUpdate(ctx, token, empNo,
		data.EmployeePartial{
			FirstName: &updatedFirstName,
			LastName:  &updatedLastName,
		})
	assert.Nil(t, err)
	assert.NotNil(t, employeeUpdated)

	// read employee
	employeeRead, err = c.EmployeeRead(ctx, token, empNo)
	assert.Nil(t, err)
	assert.NotNil(t, employeeRead)
	assert.Equal(t, employeeUpdated, employeeRead)

	// delete employee
	err = c.EmployeeDelete(ctx, token, empNo)
	assert.Nil(t, err)
}

func testClient(t *testing.T) {
	c := newClientTest()

	ctx := t.Context()
	err := c.Configure(envs)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to configure testClient")
	}
	err = c.Open(ctx)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to open testClient")
	}
	defer c.Close(ctx)
	t.Run("Client", c.TestClient)
}

func TestClient(t *testing.T) {
	testClient(t)
}
