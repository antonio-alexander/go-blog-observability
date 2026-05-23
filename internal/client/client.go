package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
)

type client struct {
	sync.RWMutex
	config struct {
		protocol   string
		address    string
		port       string
		timeout    int64
		sslCaFile  string
		sslCrtFile string
		sslKeyFile string
		maxRetries int
	}
	address   string
	ctx       context.Context
	ctxCancel context.CancelFunc
	opened    bool
	logger.Logger
	*http.Client
}

func NewClient(parameters ...any) interface {
	internal.Configurer
	internal.Opener
	Client
} {
	c := &client{Client: &http.Client{}}
	for _, parameter := range parameters {
		switch p := parameter.(type) {
		case logger.Logger:
			c.Logger = p
		}
	}
	return c
}

func (c *client) doRequest(ctx context.Context, method, uri, authorization string, item any) ([]byte, error) {
	bytes, retryAfter, err := doRequest(ctx, c.Client, method, uri, authorization, item)
	if c.config.maxRetries > 0 && retryAfter > 0 {
		c.Debug(ctx, "client attempting to retry; initial request failed",
			slog.String("uri", uri),
			slog.String("method", method),
			slog.Int("max_retries", c.config.maxRetries),
		)
		for i := range c.config.maxRetries {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("client context canclled while retrying: %w", context.Canceled)
			case <-time.After(retryAfter):
				c.Debug(ctx, "client attempting to retry",
					slog.String("uri", uri), slog.String("method", method),
					slog.Int("max_retries", c.config.maxRetries),
					slog.Int("attempt", i))
				bytes, retryAfter, err = doRequest(ctx, c.Client, uri, method, authorization, item)
				if retryAfter <= 0 {
					return bytes, err
				}
			}
		}
	}
	return bytes, err
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
		if err != nil {
			return err
		}
		c.config.timeout = i
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
	if maxRetries, ok := envs["CLIENT_MAX_RETRIES"]; ok {
		i, err := strconv.Atoi(maxRetries)
		if err != nil {
			return err
		}
		c.config.maxRetries = i
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
	c.Client.Timeout = time.Duration(c.config.timeout) * time.Second
	tlsConfig, err := getTlsConfig(c.config.sslCaFile, c.config.sslCrtFile,
		c.config.sslKeyFile)
	if err != nil {
		return err
	}
	c.Client.Transport = tlsConfig
	c.ctx, c.ctxCancel = context.WithCancel(context.Background())
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
	bytes, err := json.Marshal(&data.Request{
		EmployeePartial: employeePartial})
	if err != nil {
		return nil, err
	}
	uri := c.address + data.RouteEmployees
	bytes, err = c.doRequest(ctx, http.MethodPut, uri, token, bytes)
	if err != nil {
		return nil, err
	}
	response := &data.Response{}
	if err := json.Unmarshal(bytes, response); err != nil {
		return nil, err
	}
	return response.Employee, nil
}

func (c *client) EmployeeRead(ctx context.Context, token string, empNo int64) (*data.Employee, error) {
	uri := fmt.Sprintf(c.address+data.RouteEmployeesEmpNof, empNo)
	bytes, err := c.doRequest(ctx, http.MethodGet, uri, token, nil)
	if err != nil {
		return nil, err
	}
	response := &data.Response{}
	if err := json.Unmarshal(bytes, response); err != nil {
		return nil, err
	}
	return response.Employee, nil
}

func (c *client) EmployeesSearch(ctx context.Context, token string, search data.EmployeeSearch) ([]*data.Employee, error) {
	var response data.Response

	params := search.ToParams()
	uri := c.address + data.RouteEmployeesSearch
	bytes, err := c.doRequest(ctx, http.MethodGet, uri, token, params)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bytes, &response); err != nil {
		return nil, err
	}
	return response.Employees, nil
}

func (c *client) EmployeeUpdate(ctx context.Context, token string, empNo int64, employeePartial data.EmployeePartial) (*data.Employee, error) {
	bytes, err := json.Marshal(&data.Request{EmployeePartial: employeePartial})
	if err != nil {
		return nil, err
	}
	uri := fmt.Sprintf(c.address+data.RouteEmployeesEmpNof, empNo)
	bytes, err = c.doRequest(ctx, http.MethodPost, uri, token, bytes)
	if err != nil {
		return nil, err
	}
	response := &data.Response{}
	if err := json.Unmarshal(bytes, response); err != nil {
		return nil, err
	}
	return response.Employee, nil
}

func (c *client) EmployeeDelete(ctx context.Context, token string, empNo int64) error {
	uri := fmt.Sprintf(c.address+data.RouteEmployeesEmpNof, empNo)
	if _, err := c.doRequest(ctx, http.MethodDelete, uri, token, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) Sleep(ctx context.Context, s data.Sleep) (*data.Sleep, error) {
	uri := c.address + data.RouteSleep
	bytes, err := json.Marshal(&s)
	if err != nil {
		return nil, err
	}
	bytes, err = c.doRequest(ctx, http.MethodPost, uri, "", bytes)
	if err != nil {
		return nil, err
	}
	sleep := &data.Sleep{}
	if err := json.Unmarshal(bytes, sleep); err != nil {
		return nil, err
	}
	return sleep, nil
}

func (c *client) CacheClear(ctx context.Context) error {
	uri := c.address + data.RouteCache
	if _, err := c.doRequest(ctx, http.MethodDelete, uri, "", nil); err != nil {
		return err
	}
	return nil
}

func (c *client) Panic(ctx context.Context) error {
	uri := c.address + data.RoutePanic
	if _, err := c.doRequest(ctx, http.MethodPost, uri, "", nil); err != nil {
		return err
	}
	return nil
}
