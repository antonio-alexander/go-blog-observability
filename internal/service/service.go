package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/authz"
	"github.com/antonio-alexander/go-blog-observability/internal/cache"
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	pkgcontext "github.com/antonio-alexander/go-blog-observability/internal/pkg/context"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/tracer"

	"github.com/rs/cors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

type service struct {
	sync.RWMutex
	sync.WaitGroup
	*http.ServeMux
	*http.Server
	logger.Logger
	tracer.Tracer
	metrics.Metrics
	*metrics.Int64UpDownCounterHelper
	*metrics.Int64CounterHelper
	*metrics.Float64HistogramHelper
	*metrics.Int64HistogramHelper
	authz.Authz
	config struct {
		address          string
		port             string
		timeout          time.Duration
		shutdownTimeout  time.Duration
		allowedOrigins   []string
		allowedMethods   []string
		allowedHeaders   []string
		allowCredentials bool
		corsDisabled     bool
		corsDebug        bool
		hostname         string
		maxRequestBytes  int64
	}
	ctx    context.Context
	cancel context.CancelFunc
	cache  internal.Clearer
	opened atomic.Bool
}

func New(parameters ...any) interface {
	internal.Configurer
	internal.Opener
} {
	serveMux := http.NewServeMux()
	s := &service{
		Int64CounterHelper:       metrics.NewInt64CounterHelper(),
		Float64HistogramHelper:   metrics.NewFloat64HistogramHelper(),
		Int64HistogramHelper:     metrics.NewInt64HistogramHelper(),
		Int64UpDownCounterHelper: metrics.NewInt64UpDownCounterHelper(),
		ServeMux:                 serveMux,
		Server: &http.Server{
			Handler: serveMux,
		},
	}
	for _, parameter := range parameters {
		switch p := parameter.(type) {
		case interface {
			cache.Cache
			internal.Clearer
		}:
			s.cache = p
		case authz.Authz:
			s.Authz = p
		case metrics.Metrics:
			s.Metrics = p
		case tracer.Tracer:
			s.Tracer = p
		case logger.Logger:
			s.Logger = p
		}
	}
	switch {
	case s.Logger == nil:
		panic("logger not set")
	case s.Authz == nil:
		panic("authz not set")
	case s.Metrics == nil:
		panic("metrics not set")
	case s.Tracer == nil:
		panic("tracer not set")
	}
	return s
}

func (s *service) handleResponse(ctx context.Context, w http.ResponseWriter, err error, items ...any) {
	if err := handleResponse(w, err, items...); err != nil {
		s.Error(ctx, "unable to handle response", err)
	}
}

func (s *service) launchServer() error {
	started := make(chan struct{})
	chErr := make(chan error, 1)
	s.Go(func() {
		defer close(chErr)
		if !s.config.corsDisabled {
			s.Server.Handler = cors.New(cors.Options{
				AllowedOrigins:   s.config.allowedOrigins,
				AllowCredentials: s.config.allowCredentials,
				AllowedMethods:   s.config.allowedMethods,
				AllowedHeaders:   s.config.allowedHeaders,
				Debug:            s.config.corsDebug,
			}).Handler(s.ServeMux)
		}
		close(started)
		if err := s.ListenAndServe(); err != nil {
			chErr <- err
		}
	})
	<-started
	select {
	case err := <-chErr:
		// here we're accounting for a situation where the server closes unexexpectedly
		// but quickly (within a second of starting); this allows us to respond to errors such as
		// the port being already used
		return err
	case <-time.After(time.Second):
		address := net.JoinHostPort(s.config.address, s.config.port)
		s.Info(s.ctx, "started server", slog.String("address", address))
		return nil
	}
}

func (s *service) middlewareInitialize() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := s.ExtractTrace(r)
			ctx = pkgcontext.WithAttributes(ctx,
				slog.String("correlation_id", getCorrelationId(r)),
				slog.String("request_id", internal.GenerateId()),
				slog.String("http.method", r.Method),
				slog.String("http.route", r.Pattern),
			)
			spanName := r.Method + " " + r.Pattern
			ctx, span := s.Start(ctx, spanName)
			defer span.End()
			r = r.WithContext(ctx)
			wWrapped := &responseWriterWrapper{ResponseWriter: w}
			next.ServeHTTP(wWrapped, r)
			switch wWrapped.statusCode {
			case http.StatusOK, http.StatusNoContent:
				span.SetStatus(codes.Ok, "")
			}
		})
	}
}

func (s *service) middlewareLatency() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func(ctx context.Context, start time.Time) {
				duration := time.Since(start)
				s.Float64HistogramHelper.Record(ctx, histogramNameDuration,
					duration.Seconds())
			}(r.Context(), time.Now())
			next.ServeHTTP(w, r)
		})
	}
}

func (s *service) middlewarePanic() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func(ctx context.Context) {
				if recvr := recover(); recvr != nil {
					s.Int64CounterHelper.Add(ctx, counterNamePanicsTotal, 1)
					args := make([]slog.Attr, 0, 2)
					args = append(args, slog.String("stack_trace", string(debug.Stack())))
					args = append(args, slog.String("panic", fmt.Sprintf("%v", r)))
					if runErr, ok := recvr.(runtime.Error); ok {
						args = append(args, slog.String("runtime_error", runErr.Error()))
					}
					s.Warn(ctx, "panic has occurred", args)
				}
			}(r.Context())
			next.ServeHTTP(w, r)
		})
	}
}

func (s *service) middlewareMaxRequestSize() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, s.config.maxRequestBytes)
			next.ServeHTTP(w, r)
		})
	}
}

func (s *service) middlewareRequest() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			s.Int64UpDownCounterHelper.Add(ctx, counterNameActiveRequests, 1)
			defer s.Int64UpDownCounterHelper.Add(ctx, counterNameActiveRequests, -1)
			s.Int64CounterHelper.Add(ctx, counterNameRequestsTotal, 1)
			wWrapped := &responseWriterWrapper{ResponseWriter: w}
			s.Int64CounterHelper.Add(ctx, counterNameTotalRequestBytes, r.ContentLength)
			s.Int64HistogramHelper.Record(ctx, histogramNameRequestBytes, r.ContentLength)
			next.ServeHTTP(wWrapped, r)
			if contentLength := int64(len(wWrapped.bytes)); contentLength > 0 {
				s.Int64CounterHelper.Add(ctx, counterNameTotalResponseBytes, contentLength)
				s.Int64HistogramHelper.Record(ctx, histogramNameResponseBytes, contentLength)
			}
			option := metric.WithAttributes(
				attribute.Int("http.status_code", wWrapped.statusCode),
			)
			switch wWrapped.statusCode {
			default:
				s.Int64CounterHelper.Add(ctx, counterNameRequestsFailed, 1, option)
				if len(wWrapped.bytes) > 0 {
					var e error

					errUnauthorized := new(errors.ErrorUnauthorized)
					if err := json.Unmarshal(wWrapped.bytes, errUnauthorized); err == nil &&
						errUnauthorized.ErrorType != "" {
						e = errUnauthorized
					}
					if e == nil {
						errCommon := new(errors.ErrorCommon)
						if err := json.Unmarshal(wWrapped.bytes, errCommon); err == nil &&
							errCommon.ErrorType != "" {
							e = errCommon
						}
					}
					s.Error(ctx, "request failed", slog.Any("error", e))
				}
			case http.StatusOK, http.StatusNoContent:
				s.Int64CounterHelper.Add(ctx, counterNameRequestsSuccessful, 1, option)
			}
		})
	}
}

func (s *service) endpointDefault(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprintf(w,
		"go-blog-observability\n"+
			"Version: \"%s\"\n"+
			"Git Commit: \"%s\"\n"+
			"Git Branch: \"%s\"\n",
		data.Version, data.GitCommit, data.GitBranch)
}

func (s *service) endpointAuthorize(w http.ResponseWriter, r *http.Request) {
	var userId string

	ctx := r.Context()
	if _, err := readRequestBody(r, &struct {
		UserId *string `json:"user_id"`
	}{
		UserId: &userId,
	}); err != nil {
		s.handleResponse(ctx, w, err, nil)
		return
	}
	defer func() {
		_ = r.Body.Close()
	}()
	authorization, err := s.Authorize(ctx, userId)
	if err != nil {
		s.handleResponse(r.Context(), w, err, nil)
		return
	}
	s.handleResponse(ctx, w, err, &data.Response{
		Authorization: &authorization,
	})
}

func (s *service) endpointEmployeeCreate(w http.ResponseWriter, r *http.Request) {
	var request data.Request

	ctx := r.Context()
	if _, err := readRequestBody(r, &request); err != nil {
		s.handleResponse(r.Context(), w, err, nil)
		return
	}
	authorization := getAuthorization(r)
	employee, err := s.EmployeeCreate(ctx, authorization, request.EmployeePartial)
	s.handleResponse(r.Context(), w, err, &data.Response{
		Employee: employee,
	})
}

func (s *service) endpointEmployeeRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	empNo, err := strconv.ParseInt(r.PathValue(data.PathEmpNo), 10, 64)
	if err != nil {
		s.handleResponse(r.Context(), w, err, nil)
		return
	}
	authorization := getAuthorization(r)
	employee, err := s.EmployeeRead(ctx, authorization, empNo)
	s.handleResponse(r.Context(), w, err,
		&data.Response{Employee: employee})
}

func (s *service) endpointEmployeesSearch(w http.ResponseWriter, r *http.Request) {
	var search data.EmployeeSearch

	ctx := r.Context()
	switch r.Method {
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	case "QUERY":
		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			s.handleResponse(r.Context(), w, err, nil)
			return
		}
		if err := json.Unmarshal(bytes, &search); err != nil {
			s.handleResponse(r.Context(), w, err, nil)
			return
		}
	case http.MethodGet:
		if err := r.ParseForm(); err != nil {
			s.handleResponse(r.Context(), w, err, nil)
			return
		}
		search.FromParams(r.Form)
	}
	authorization := getAuthorization(r)
	employees, err := s.EmployeesSearch(ctx, authorization, search)
	s.handleResponse(r.Context(), w, err, &data.Response{Employees: employees})
}

func (s *service) endpointEmployeeUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	empNo, err := strconv.ParseInt(r.PathValue(data.PathEmpNo), 10, 64)
	if err != nil {
		s.handleResponse(r.Context(), w, err, nil)
		return
	}
	request := &data.Request{}
	if _, err := readRequestBody(r, request); err != nil {
		s.handleResponse(r.Context(), w, err, nil)
		return
	}
	authorization := getAuthorization(r)
	employee, err := s.EmployeeUpdate(ctx, authorization,
		empNo, request.EmployeePartial)
	s.handleResponse(r.Context(), w, err, &data.Response{Employee: employee})
}

func (s *service) endpointEmployeeDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	empNo, err := strconv.ParseInt(r.PathValue(data.PathEmpNo), 10, 64)
	if err != nil {
		s.handleResponse(r.Context(), w, err, nil)
		return
	}
	authorization := getAuthorization(r)
	err = s.EmployeeDelete(ctx, authorization, empNo)
	s.handleResponse(r.Context(), w, err, nil)
}

func (s *service) endpointSleep(w http.ResponseWriter, r *http.Request) {
	var request data.Sleep

	ctx := r.Context()
	if _, err := readRequestBody(r, &request); err != nil {
		s.handleResponse(r.Context(), w, err, nil)
		return
	}
	sleep, err := s.Sleep(ctx, request)
	s.handleResponse(r.Context(), w, err, sleep)
}

func (s *service) endpointPanic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s.Panic(ctx)
}

func (s *service) endpointCacheClear(w http.ResponseWriter, r *http.Request) {
	if s.cache != nil {
		ctx := r.Context()
		if err := s.cache.Clear(ctx); err != nil {
			s.handleResponse(r.Context(), w, err, nil)
			return
		}
	}
	s.handleResponse(r.Context(), w, nil, nil)
}

func (s *service) handleFunc(path string, f http.HandlerFunc, middlewares ...middleware) {
	if len(middlewares) <= 0 {
		s.HandleFunc(path, f)
		return
	}
	s.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		final := http.Handler(f)
		for _, middleware := range slices.Backward(middlewares) {
			final = middleware(final)
		}
		final.ServeHTTP(w, r)
	})
}

func (s *service) buildRoutes() {
	//generate middleware
	middlewares := []middleware{
		s.middlewareInitialize(),
		s.middlewareMaxRequestSize(),
		s.middlewareLatency(),
		s.middlewarePanic(),
		s.middlewareRequest(),
	}

	//build routes
	s.handleFunc("/", s.endpointDefault, middlewares...)
	s.handleFunc(data.RouteSleep, s.endpointSleep, middlewares...)
	s.handleFunc(data.RoutePanic, s.endpointPanic, middlewares...)
	s.handleFunc(data.RouteEmployeesSearch, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		case http.MethodGet, "QUERY":
			s.endpointEmployeesSearch(w, r)
		}
	}, middlewares...)
	s.handleFunc(data.RouteEmployees,
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			case http.MethodPut:
				s.endpointEmployeeCreate(w, r)
			}
		}, middlewares...)
	s.handleFunc(data.RouteEmployeesEmpNo,
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			case http.MethodGet:
				s.endpointEmployeeRead(w, r)
			case http.MethodPost:
				s.endpointEmployeeUpdate(w, r)
			case http.MethodDelete:
				s.endpointEmployeeDelete(w, r)
			}
		}, middlewares...)
	s.handleFunc(data.RouteCache,
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			case http.MethodDelete:
				s.endpointCacheClear(w, r)
			}
		}, middlewares...)
	s.handleFunc(data.RouteAuthorize, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		case http.MethodPost:
			s.endpointAuthorize(w, r)
		}
	}, middlewares...)
}

func (s *service) Configure(envs map[string]string) error {
	//set defaults
	s.config.maxRequestBytes = 1048576 //1MB

	//set configuration
	if address, ok := envs["SERVICE_ADDRESS"]; ok {
		s.config.address = address
	}
	if port, ok := envs["SERVICE_PORT"]; ok {
		s.config.port = port
	}
	if shutdownTimeoutString, ok := envs["SERVICE_SHUTDOWN_TIMEOUT"]; ok {
		if shutdownTimeoutInt, err := strconv.Atoi(shutdownTimeoutString); err == nil {
			if timeout := time.Duration(shutdownTimeoutInt) * time.Second; timeout > 0 {
				s.config.shutdownTimeout = timeout
			}
		}
	}
	if allowCredentialsString, ok := envs["SERVICE_CORS_ALLOW_CREDENTIALS"]; ok {
		if allowCredentials, err := strconv.ParseBool(allowCredentialsString); err == nil {
			s.config.allowCredentials = allowCredentials
		}
	}
	if allowedOrigins, ok := envs["SERVICE_CORS_ALLOWED_ORIGINS"]; ok {
		s.config.allowedOrigins = strings.Split(allowedOrigins, ",")
	}
	if allowedMethods, ok := envs["SERVICE_CORS_ALLOWED_METHODS"]; ok {
		s.config.allowedMethods = strings.Split(allowedMethods, ",")
	}
	if allowedHeaders, ok := envs["SERVICE_CORS_ALLOWED_HEADERS"]; ok {
		s.config.allowedHeaders = strings.Split(allowedHeaders, ",")
	}
	if corsDisabledString, ok := envs["SERVICE_CORS_DISABLED"]; ok {
		if corsDisabled, err := strconv.ParseBool(corsDisabledString); err == nil {
			s.config.corsDisabled = corsDisabled
		}
	}
	if corsDebug, ok := envs["SERVICE_CORS_DEBUG"]; ok {
		if corsDebug, err := strconv.ParseBool(corsDebug); err == nil {
			s.config.corsDebug = corsDebug
		}
	}
	if hostname, ok := envs["HOSTNAME"]; ok {
		s.config.hostname = hostname
	}
	if maxRequestBytes, ok := envs["SERVICE_MAX_REQUEST_BYTES"]; ok {
		if i, err := strconv.ParseInt(maxRequestBytes, 10, 64); err == nil {
			s.config.maxRequestBytes = i
		}
	}
	return nil
}

func (s *service) Open(ctx context.Context) error {
	if s.opened.Load() {
		return nil
	}
	meter := s.Meter(packageName)
	if err := createCounters(meter, s.Int64CounterHelper); err != nil {
		return err
	}
	if err := createUpDownCounters(meter, s.Int64UpDownCounterHelper); err != nil {
		return err
	}
	if err := createHistograms(meter, s.Float64HistogramHelper, s.Int64HistogramHelper); err != nil {
		return err
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.Addr = net.JoinHostPort(s.config.address, s.config.port)
	s.buildRoutes()
	if err := s.launchServer(); err != nil {
		return err
	}
	s.opened.Store(true)
	return nil
}

func (s *service) Close(ctx context.Context) {
	if s.opened.Load() {
		return
	}
	s.opened.Store(false)
	ctx, cancel := context.WithTimeout(ctx, s.config.shutdownTimeout)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		s.Error(ctx, "unable to shutdown rest service", err)
	}
	s.cancel()
	s.Wait()
	s.Float64HistogramHelper.Close(ctx)
	s.Int64CounterHelper.Close(ctx)
	s.Int64UpDownCounterHelper.Close(ctx)
}
