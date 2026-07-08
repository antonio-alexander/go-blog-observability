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
	"strconv"
	"strings"
	"sync"
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

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type service struct {
	sync.RWMutex
	sync.WaitGroup
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
	logger.Logger
	metrics.Metrics
	tracer.Tracer
	authz.Authz
	*mux.Router
	*http.Server
	meter      metrics.Meter
	histograms struct {
		sync.RWMutex
		data map[string]metrics.Float64Histogram
	}
	counters struct {
		sync.RWMutex
		data map[string]metrics.Int64UpDownCounter
	}
}

func New(parameters ...any) interface {
	internal.Configurer
	internal.Opener
} {
	router := mux.NewRouter()
	s := &service{
		Router: router,
		Server: &http.Server{
			Handler: router,
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
		panic("Logger not set")
	case s.Authz == nil:
		panic("Authz not set")
	case s.Metrics == nil:
		panic("Metrics not set")
	case s.Tracer == nil:
		panic("Tracer not set")
	}
	s.counters.data = make(map[string]metrics.Int64UpDownCounter)
	s.histograms.data = make(map[string]metrics.Float64Histogram)
	return s
}

func (s *service) launchServer() error {
	started := make(chan struct{})
	chErr := make(chan error, 1)
	s.Add(1)
	go func() {
		defer s.WaitGroup.Done()
		defer close(chErr)

		if !s.config.corsDisabled {
			s.Server.Handler = cors.New(cors.Options{
				AllowedOrigins:   s.config.allowedOrigins,
				AllowCredentials: s.config.allowCredentials,
				AllowedMethods:   s.config.allowedMethods,
				AllowedHeaders:   s.config.allowedHeaders,
				Debug:            s.config.corsDebug,
			}).Handler(s.Router)
		}
		close(started)
		if err := s.Server.ListenAndServe(); err != nil {
			chErr <- err
		}
	}()
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

func (s *service) readRequestBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, int64(s.config.maxRequestBytes))
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		switch {
		default:
			return nil, err
		case errors.Is(err, &http.MaxBytesError{}):
			return nil, &errors.ErrorCommon{
				Err:                err,
				ErrorMessage:       err.Error(),
				ErrorMessageDetail: "",
				ErrorType:          errors.ErrorTypeTooMuchData,
				Local:              true,
			}
		}
	}
	return bytes, nil
}

func (s *service) createHistogram(histogramName string) (metrics.Float64Histogram, error) {
	s.histograms.Lock()
	defer s.histograms.Unlock()
	histogram, err := createHistogram(s.meter, histogramName)
	if err != nil {
		return nil, err
	}
	s.histograms.data[histogramName] = histogram
	return histogram, nil
}

func (s *service) createCounter(counterName string) (metrics.Int64UpDownCounter, error) {
	s.counters.Lock()
	defer s.counters.Unlock()
	counter, err := createCounter(s.meter, counterName)
	if err != nil {
		return nil, err
	}
	s.counters.data[counterName] = counter
	return counter, nil
}

func (s *service) readSpanCounter(spanName string) (metrics.Int64UpDownCounter, error) {
	s.counters.Lock()
	defer s.counters.Unlock()
	counter, ok := s.counters.data[spanName]
	if !ok {
		c, err := s.meter.Int64UpDownCounter(spanName)
		if err != nil {
			return nil, err
		}
		counter = c
		s.counters.data[spanName] = c
	}
	return counter, nil
}

func (s *service) readCounter(counterName string) metrics.Int64UpDownCounter {
	s.counters.RLock()
	defer s.counters.RUnlock()
	counter, ok := s.counters.data[counterName]
	if !ok {
		return nil
	}
	return counter
}

func (s *service) readHistogram(histogramName string) metrics.Float64Histogram {
	s.histograms.RLock()
	defer s.histograms.RUnlock()
	histogram, ok := s.histograms.data[histogramName]
	if !ok {
		return nil
	}
	return histogram
}

func (s *service) middlewareInitialize() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := pkgcontext.WithCorrelationId(r.Context(), getCorrelationId(r))
			ctx = pkgcontext.WithRequestId(ctx, internal.GenerateId())
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func (s *service) middlewareLatency() middleware {
	histogramDuration := s.readHistogram(histogramNameDuration)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func(ctx context.Context, start time.Time) {
				histogramDuration.Record(ctx, time.Since(start).Seconds(),
					metric.WithAttributes(
						attribute.String("http.method", r.Method),
						attribute.String("http.route", routeFromPath(r.URL.Path)),
					))
			}(r.Context(), time.Now())
			next.ServeHTTP(w, r)
		})
	}
}

func (s *service) middlewarePanic() middleware {
	panicCounter := s.readCounter(counterNamePanicsTotal)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func(ctx context.Context) {
				if r := recover(); r != nil {
					panicCounter.Add(ctx, 1)
					args := make([]slog.Attr, 0, 2)
					args = append(args, slog.String("stack_trace", string(debug.Stack())))
					args = append(args, slog.String("panic", fmt.Sprintf("%v", r)))
					if runErr, ok := r.(runtime.Error); ok {
						args = append(args, slog.String("runtime_error", runErr.Error()))
					}
					s.Warn(ctx, "panic has occurred", args)
				}
			}(r.Context())
			next.ServeHTTP(w, r)
		})
	}
}

func (s *service) middlewareRequest() middleware {
	counterActiveRequests := s.readCounter(counterNameActiveRequests)
	counterRequestsTotal := s.readCounter(counterNameRequestsTotal)
	counterTotalRequestBytes := s.readCounter(counterNameTotalRequestBytes)
	counterResponseBytes := s.readCounter(counterNameTotalResponseBytes)
	counterFailedRequests := s.readCounter(counterNameRequestsFailed)
	counterSuccessfulRequests := s.readCounter(counterNameRequestsSuccessful)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			counterActiveRequests.Add(ctx, 1) //active requests
			defer counterActiveRequests.Add(ctx, -1)
			counterRequestsTotal.Add(ctx, 1) //total requests
			statusWriter := &statusCodeResponseWriter{ResponseWriter: w}
			counterTotalRequestBytes.Add(ctx, r.ContentLength)                          //request bytes
			next.ServeHTTP(statusWriter, r)                                             //execute next request
			if contentLength := w.Header().Get("Content-Length"); contentLength != "" { //get response length
				i, err := strconv.ParseInt(contentLength, 10, 64)
				if err == nil {
					counterResponseBytes.Add(ctx, i)
				}
			}
			switch statusWriter.statusCode {
			default:
				counterFailedRequests.Add(r.Context(), 1)
			case http.StatusOK, http.StatusNoContent:
				counterSuccessfulRequests.Add(r.Context(), 1)
			}
		})
	}
}

func (s *service) middlewareSpan() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			spanName := fmt.Sprintf("%s_%s", r.Method, routeFromPath(r.URL.Path))
			ctx, span := s.Start(r.Context(), spanName, trace.WithNewRoot())
			defer span.End()
			if spanCounter, err := s.readSpanCounter(spanName); err == nil {
				spanCounter.Add(ctx, 1, metric.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.route", routeFromPath(r.URL.Path)),
				))
			}
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func (s *service) endpointDefault(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer,
		"go-blog-observability\n"+
			"Version: \"%s\"\n"+
			"Git Commit: \"%s\"\n"+
			"Git Branch: \"%s\"\n",
		data.Version, data.GitCommit, data.GitBranch)
}

func (s *service) endpointEmployeeCreate(writer http.ResponseWriter, request *http.Request) {
	var employeeRequest data.Request

	ctx := request.Context()
	bytes, err := s.readRequestBody(writer, request)
	defer request.Body.Close()
	if err != nil {
		_ = handleResponse(writer, err, nil)
		return
	}
	if err := json.Unmarshal(bytes, &employeeRequest); err != nil {
		_ = handleResponse(writer, err, nil)
		return
	}
	employee, err := s.EmployeeCreate(ctx, getAuthorization(request), employeeRequest.EmployeePartial)
	if err := handleResponse(writer, err, &data.Response{
		Employee: employee,
	}); err != nil {
		s.Error(ctx, "unable to handle response", err)
		return
	}
	if err == nil {
		s.Debug(ctx, "successfully executed employee_create", slog.Int64("emp_no", employee.EmpNo))
	} else {
		s.Debug(ctx, "failed to execute employee_create", err, slog.Int64("emp_no", employee.EmpNo))
	}
}

func (s *service) endpointEmployeeRead(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	empNo, err := empNoFromPath(mux.Vars(request))
	if err != nil {
		_ = handleResponse(writer, err, nil)
		return
	}
	employee, err := s.EmployeeRead(ctx, getAuthorization(request), empNo)
	if err := handleResponse(writer, err, &data.Response{
		Employee: employee,
	}); err != nil {
		s.Error(ctx, "unable to handle response", err)
		return
	}
	if err == nil {
		s.Debug(ctx, "successfully executed employee_read", slog.Int64("emp_no", empNo))
	} else {
		s.Debug(ctx, "failed to execute employee_read", err, slog.Int64("emp_no", empNo))
	}
}

func (s *service) endpointEmployeesSearch(writer http.ResponseWriter, request *http.Request) {
	var search data.EmployeeSearch

	ctx := request.Context()
	if err := request.ParseForm(); err != nil {
		_ = handleResponse(writer, err, nil)
		return
	}
	search.FromParams(request.Form)
	employees, err := s.EmployeesSearch(ctx, getAuthorization(request), search)
	if err := handleResponse(writer, err, &data.Response{
		Employees: employees,
	}); err != nil {
		s.Error(ctx, "unable to handle response", err)
		return
	}
	if err == nil {
		s.Debug(ctx, "successfully executed employees_search")
	} else {
		s.Debug(ctx, "failed to execute employees_search", err)
	}
}

func (s *service) endpointEmployeeUpdate(writer http.ResponseWriter, request *http.Request) {
	var employeeRequest data.Request

	ctx := request.Context()
	empNo, err := empNoFromPath(mux.Vars(request))
	if err != nil {
		_ = handleResponse(writer, err, nil)
		return
	}
	bytes, err := s.readRequestBody(writer, request)
	defer request.Body.Close()
	if err != nil {
		_ = handleResponse(writer, err, nil)
		return
	}
	if err := json.Unmarshal(bytes, &employeeRequest); err != nil {
		_ = handleResponse(writer, err, nil)
		return
	}
	employee, err := s.EmployeeUpdate(ctx, getAuthorization(request),
		empNo, employeeRequest.EmployeePartial)
	if err := handleResponse(writer, err, &data.Response{
		Employee: employee,
	}); err != nil {
		s.Error(ctx, "unable to handle response", err)
		return
	}
	if err == nil {
		s.Debug(ctx, "successfully executed employee_update", slog.Int64("emp_no", employee.EmpNo))
	} else {
		s.Debug(ctx, "failed to execute employee_update", slog.Int64("emp_no", employee.EmpNo))
	}
}

func (s *service) endpointEmployeeDelete(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	empNo, err := empNoFromPath(mux.Vars(request))
	if err != nil {
		_ = handleResponse(writer, err, nil)
		return
	}
	err = s.EmployeeDelete(ctx, getAuthorization(request), empNo)
	if err := handleResponse(writer, err, nil); err != nil {
		s.Error(ctx, "unable to handle response", err)
		return
	}
	if err == nil {
		s.Debug(ctx, "successfully executed employee_delete", slog.Int64("emp_no", empNo))
	} else {
		s.Debug(ctx, "failed to execute employee_delete", err, slog.Int64("emp_no", empNo))
	}
}

func (s *service) endpointSleep(writer http.ResponseWriter, request *http.Request) {
	var sleepRequest data.Sleep

	ctx := request.Context()
	bytes, err := s.readRequestBody(writer, request)
	defer request.Body.Close()
	if err != nil {
		_ = handleResponse(writer, err, nil)
		return
	}
	if err := json.Unmarshal(bytes, &sleepRequest); err != nil {
		_ = handleResponse(writer, err, nil)
		return
	}
	sleep, err := s.Sleep(ctx, sleepRequest)
	if err := handleResponse(writer, err, sleep); err != nil {
		s.Error(ctx, "unable to handle response", err)
		return
	}
	if err == nil {
		s.Debug(ctx, "successfully executed sleep", slog.String("sleep_id", sleep.Id),
			slog.Duration("sleep_duration", time.Duration(sleep.Duration)))
	} else {
		s.Debug(ctx, "failed to execute sleep", err, slog.String("sleep_id", sleepRequest.Id))
	}
}

func (s *service) endpointPanic(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	s.Panic(ctx)
}

func (s *service) endpointCacheClear(writer http.ResponseWriter, request *http.Request) {
	if s.cache != nil {
		ctx := request.Context()
		if err := s.cache.Clear(ctx); err != nil {
			_ = handleResponse(writer, err, nil)
			return
		}
		s.Debug(ctx, "executed cache_clear")
	}
	_ = handleResponse(writer, nil, nil)
}

func (s *service) handleFunc(path string, f http.HandlerFunc, middlewares ...middleware) *mux.Route {
	if len(middlewares) <= 0 {
		return s.Router.HandleFunc(path, f)
	}
	return s.Router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		final := http.Handler(f)
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		final.ServeHTTP(w, r)
	})
}

func (s *service) buildRoutes() {
	//generate middleware
	middlewares := []middleware{
		s.middlewareInitialize(),
		s.middlewareLatency(),
		s.middlewarePanic(),
		s.middlewareRequest(),
		s.middlewareSpan(),
	}

	//build routes
	s.handleFunc("/", s.endpointDefault, middlewares...)
	s.handleFunc(data.RouteSleep, s.endpointSleep, middlewares...)
	s.handleFunc(data.RoutePanic, s.endpointPanic, middlewares...)
	s.handleFunc(data.RouteEmployeesSearch, s.endpointEmployeesSearch, middlewares...)
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
	s.Lock()
	defer s.Unlock()

	//generate endpoint counters
	meter := s.Meter(packageName)
	s.meter = meter
	for _, counterName := range counterNames {
		if _, err := s.createCounter(counterName); err != nil {
			return err
		}
	}
	for _, histogramName := range histogramNames {
		if _, err := s.createHistogram(histogramName); err != nil {
			return err
		}
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.Server.Addr = net.JoinHostPort(s.config.address, s.config.port)
	s.buildRoutes()
	return s.launchServer()
}

func (s *service) Close(ctx context.Context) {
	s.Lock()
	defer s.Unlock()

	ctx, cancel := context.WithTimeout(ctx, s.config.shutdownTimeout)
	defer cancel()
	if err := s.Server.Shutdown(ctx); err != nil {
		s.Error(ctx, "unable to shutdown rest service", err)
	}
	s.cancel()
	s.Wait()
}
