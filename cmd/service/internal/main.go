package internal

import (
	"context"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/authz"
	"github.com/antonio-alexander/go-blog-observability/internal/cache"
	"github.com/antonio-alexander/go-blog-observability/internal/logic"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/policy"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/tracer"
	"github.com/antonio-alexander/go-blog-observability/internal/service"
	"github.com/antonio-alexander/go-blog-observability/internal/sql"
)

func createLogger(envs map[string]string, parameters ...any) (internal.Opener, error) {
	switch envs["LOG_TYPE"] {
	default:
		logger := logger.NewNull(parameters...)
		if err := logger.Configure(envs); err != nil {
			return nil, err
		}
		return logger, nil
	case "slog":
		logger := logger.NewSlog(parameters...)
		if err := logger.Configure(envs); err != nil {
			return nil, err
		}
		return logger, nil
	case "open_telemetry":
		logger := logger.NewOpenTelemetry(parameters...)
		if err := logger.Configure(envs); err != nil {
			return nil, err
		}
		return logger, nil
	}
}

func createMetrics(envs map[string]string, parameters ...any) (internal.Opener, error) {
	switch envs["METRIC_TYPE"] {
	default:
		metrics := metrics.NewNull(parameters...)
		if err := metrics.Configure(envs); err != nil {
			return nil, err
		}
		return metrics, nil
	case "open_telemetry":
		metrics := metrics.NewOpenTelemetry(parameters...)
		if err := metrics.Configure(envs); err != nil {
			return nil, err
		}
		return metrics, nil
	}
}

func createTracer(envs map[string]string, parameters ...any) (internal.Opener, error) {
	switch envs["TRACER_TYPE"] {
	default:
		tracer := tracer.NewNull(parameters...)
		if err := tracer.Configure(envs); err != nil {
			return nil, err
		}
		return tracer, nil
	case "open_telemetry":
		tracer := tracer.NewOpenTelemetry(parameters...)
		if err := tracer.Configure(envs); err != nil {
			return nil, err
		}
		return tracer, nil
	}
}

func setup(envs map[string]string) ([]internal.Opener, error) {
	logger, err := createLogger(envs)
	if err != nil {
		return nil, err
	}
	metrics, err := createMetrics(envs)
	if err != nil {
		return nil, err
	}
	tracer, err := createTracer(envs)
	if err != nil {
		return nil, err
	}
	policy := policy.New(logger, tracer, metrics)
	if err := policy.Configure(envs); err != nil {
		return nil, err
	}
	sql := sql.New(logger, tracer, metrics)
	if err := sql.Configure(envs); err != nil {
		return nil, err
	}
	cache := cache.NewRedis(logger, tracer)
	if err := cache.Configure(envs); err != nil {
		return nil, err
	}
	logic := logic.NewLogic(logger, tracer,
		metrics, sql, cache)
	if err := logic.Configure(envs); err != nil {
		return nil, err
	}
	authz := authz.New(logger, tracer, metrics,
		logic, policy)
	if err := authz.Configure(envs); err != nil {
		return nil, err
	}
	service := service.New(authz,
		cache, logger, tracer, metrics)
	if err := service.Configure(envs); err != nil {
		return nil, err
	}
	return []internal.Opener{
		logger,
		metrics,
		tracer,
		cache,
		sql,
		logic,
		policy,
		authz,
		service,
	}, nil
}

func Main(ctx context.Context, pwd string, args []string, envs map[string]string) error {
	//setup parameters
	parameters, err := setup(envs)
	if err != nil {
		return err
	}

	//open and defer close
	for _, parameter := range parameters {
		if err := parameter.Open(ctx); err != nil {
			return err
		}
		defer parameter.Close(ctx)
	}

	<-ctx.Done()
	return nil
}
