package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	pkgcontext "github.com/antonio-alexander/go-blog-observability/internal/pkg/context"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/policy"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/tracer"
)

type client struct {
	sync.RWMutex
	logger.Logger
	tracer.Tracer
	config struct {
		protocol   string
		address    string
		port       string
		timeout    time.Duration
		sslCaFile  string
		sslCrtFile string
		sslKeyFile string
	}
	address   string
	ctx       context.Context
	ctxCancel context.CancelFunc
	opened    bool
	*http.Client
}

func NewClient(parameters ...any) interface {
	internal.Configurer
	internal.Opener
	Client
} {
	c := &client{
		Client: &http.Client{Timeout: 10 * time.Second},
		Tracer: tracer.NewNull(),
		Logger: logger.NewNull(),
	}
	for _, parameter := range parameters {
		switch p := parameter.(type) {
		case tracer.Tracer:
			c.Tracer = p
		case logger.Logger:
			c.Logger = p
		}
	}
	return c
}

func (c *client) doRequest(ctx context.Context, method, uri, authorization string, item any) ([]byte, error) {
	var contentType string
	var contentLength int
	var body io.Reader

	switch d := item.(type) {
	case []byte:
		body = bytes.NewBuffer(d)
		contentLength = len(d)
		contentType = "application/json"
	case url.Values:
		switch method {
		default:
			uri = uri + "?" + d.Encode()
		case http.MethodPut, http.MethodPost, http.MethodPatch:
			body = strings.NewReader(d.Encode())
			contentType = "application/x-www-form-urlencoded"
			contentLength = len(d.Encode())
		}
	}
	request, err := http.NewRequestWithContext(ctx, method, uri, body)
	if err != nil {
		return nil, err
	}
	c.PropagateTrace(ctx, request)
	if contentType != "" {
		request.Header.Add("Content-Type", contentType)
	}
	if contentLength > 0 {
		request.Header.Add("Content-Length", strconv.Itoa(contentLength))
	}
	if correlationId := pkgcontext.CorrelationIdFrom(ctx); correlationId != "" {
		request.Header.Add("Correlation-Id", pkgcontext.CorrelationIdFrom(ctx))
	}
	if authorization != "" {
		request.Header.Add("Authorization", authorization)
	}
	response, err := c.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, ErrTimeout
		}
		return nil, err
	}
	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = response.Body.Close()
	}()
	switch response.StatusCode {
	default:
		var e struct {
			ErrorType errors.ErrorType `json:"error_type"`
		}
		var err error

		if jsonErr := json.Unmarshal(bytes, &e); jsonErr == nil {
			switch e.ErrorType {
			default:
				err = errors.ErrorCommon{}
			case errors.ErrorTypeUnauthorized:
				err = errors.ErrorUnauthorized{}
			case policy.ErrorTypeRego:
				err = policy.ErrorRego{}

			}
		}
		if err := json.Unmarshal(bytes, &err); err != nil {
			return nil, &errors.ErrorCommon{
				ErrorMessage: fmt.Sprintf("status code: %d; unknown error occurred:%s",
					response.StatusCode, string(bytes)),
				ErrorType: errors.ErrorTypeUnknown,
			}
		}
		return nil, err
	case http.StatusOK, http.StatusNoContent:
		return bytes, nil
	}
}

func (c *client) Configure(envs map[string]string) error {
	if address, ok := envs["CLIENT_ADDRESS"]; ok {
		c.config.address = address
	}
	if port, ok := envs["CLIENT_PORT"]; ok {
		c.config.port = port
	}
	if protocol, ok := envs["CLIENT_PROTOCOL"]; ok {
		c.config.protocol = protocol
	}
	if timeout, ok := envs["CLIENT_TIMEOUT"]; ok {
		i, err := strconv.ParseInt(timeout, 10, 64)
		if err == nil {
			c.config.timeout = time.Duration(i) * time.Second
		}
	}
	if sslCaFile, ok := envs["SSL_CA_FILE"]; ok {
		c.config.sslCaFile = sslCaFile
	}
	if sslKeyFile, ok := envs["SSL_KEY_FILE"]; ok {
		c.config.sslKeyFile = sslKeyFile
	}
	if sslCrtFile, ok := envs["SSL_CRT_FILE"]; ok {
		c.config.sslCrtFile = sslCrtFile
	}
	return nil
}

func (c *client) Open(ctx context.Context) error {
	c.Lock()
	defer c.Unlock()

	if c.opened {
		return nil
	}
	switch c.config.protocol {
	default:
		return fmt.Errorf("unsupported protocol: %s", c.config.protocol)
	case "http", "https":
		c.address = fmt.Sprintf("%s://%s", c.config.protocol,
			net.JoinHostPort(c.config.address, c.config.port))
	}
	c.Timeout = c.config.timeout
	tlsConfig, err := getTlsConfig(c.config.sslCaFile, c.config.sslCrtFile,
		c.config.sslKeyFile)
	if err != nil {
		return err
	}
	c.Transport = tlsConfig
	c.ctx, c.ctxCancel = context.WithCancel(ctx)
	c.opened = true
	return nil
}

func (c *client) Close(ctx context.Context) {
	c.Lock()
	defer c.Unlock()

	if !c.opened {
		return
	}
	c.ctxCancel()
	c.opened = false
}

func (c *client) EmployeeCreate(ctx context.Context, token string, employeePartial data.EmployeePartial) (*data.Employee, error) {
	ctx, span := c.Start(ctx, "client.employee_create")
	defer span.End()
	bytes, err := json.Marshal(&data.Request{
		EmployeePartial: employeePartial})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	uri := c.address + data.RouteEmployees
	bytes, err = c.doRequest(ctx, http.MethodPut, uri, token, bytes)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	response := &data.Response{}
	if err := json.Unmarshal(bytes, response); err != nil {
		span.RecordError(err)
		return nil, err
	}
	return response.Employee, nil
}

func (c *client) EmployeeRead(ctx context.Context, token string, empNo int64) (*data.Employee, error) {
	ctx, span := c.Start(ctx, "client.employee_read")
	defer span.End()
	uri := fmt.Sprintf(c.address+data.RouteEmployeesEmpNof, empNo)
	bytes, err := c.doRequest(ctx, http.MethodGet, uri, token, nil)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	response := &data.Response{}
	if err := json.Unmarshal(bytes, response); err != nil {
		span.RecordError(err)
		return nil, err
	}
	return response.Employee, nil
}

func (c *client) EmployeesSearch(ctx context.Context, token string, search data.EmployeeSearch) ([]*data.Employee, error) {
	var response data.Response

	ctx, span := c.Start(ctx, "client.employees_search")
	defer span.End()
	params := search.ToParams()
	uri := c.address + data.RouteEmployeesSearch
	bytes, err := c.doRequest(ctx, http.MethodGet, uri, token, params)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	if err := json.Unmarshal(bytes, &response); err != nil {
		span.RecordError(err)
		return nil, err
	}
	return response.Employees, nil
}

func (c *client) EmployeeUpdate(ctx context.Context, token string, empNo int64, employeePartial data.EmployeePartial) (*data.Employee, error) {
	ctx, span := c.Start(ctx, "client.employee_update")
	defer span.End()
	bytes, err := json.Marshal(&data.Request{EmployeePartial: employeePartial})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	uri := fmt.Sprintf(c.address+data.RouteEmployeesEmpNof, empNo)
	bytes, err = c.doRequest(ctx, http.MethodPost, uri, token, bytes)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	response := &data.Response{}
	if err := json.Unmarshal(bytes, response); err != nil {
		span.RecordError(err)
		return nil, err
	}
	return response.Employee, nil
}

func (c *client) EmployeeDelete(ctx context.Context, token string, empNo int64) error {
	ctx, span := c.Start(ctx, "client.employee_delete")
	defer span.End()
	uri := fmt.Sprintf(c.address+data.RouteEmployeesEmpNof, empNo)
	if _, err := c.doRequest(ctx, http.MethodDelete, uri, token, nil); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (c *client) Sleep(ctx context.Context, s data.Sleep) (*data.Sleep, error) {
	ctx, span := c.Start(ctx, "client.sleep")
	defer span.End()
	uri := c.address + data.RouteSleep
	bytes, err := json.Marshal(&s)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	bytes, err = c.doRequest(ctx, http.MethodPost, uri, "", bytes)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	sleep := &data.Sleep{}
	if err := json.Unmarshal(bytes, sleep); err != nil {
		span.RecordError(err)
		return nil, err
	}
	return sleep, nil
}

func (c *client) CacheClear(ctx context.Context) error {
	ctx, span := c.Start(ctx, "client.cache_clear")
	defer span.End()
	uri := c.address + data.RouteCache
	if _, err := c.doRequest(ctx, http.MethodDelete, uri, "", nil); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (c *client) Panic(ctx context.Context) error {
	ctx, span := c.Start(ctx, "client.panic")
	defer span.End()
	uri := c.address + data.RoutePanic
	if _, err := c.doRequest(ctx, http.MethodPost, uri, "", nil); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (c *client) Authorize(ctx context.Context, userId string) (string, error) {
	ctx, span := c.Start(ctx, "client.authorize")
	defer span.End()
	uri := c.address + data.RouteAuthorize
	bytes, err := c.doRequest(ctx, http.MethodPost, uri, "",
		&data.Request{UserId: userId})
	if err != nil {
		span.RecordError(err)
		return "", err
	}
	response := &data.Response{}
	if err := json.Unmarshal(bytes, response); err != nil {
		return "", err
	}
	if response.Authorization != nil && *response.Authorization != "" {
		return *response.Authorization, nil
	}
	return "", errors.Must(errors.New("no authorization returned"))
}
