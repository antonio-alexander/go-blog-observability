package logger_test

import (
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"

	"github.com/google/uuid"
)

var envs = map[string]string{
	"LOG_LEVEL": "trace",
}

func init() {
	envs := make(map[string]string)
	for _, env := range os.Environ() {
		if key, value, ok := strings.Cut(env, "="); ok && value != "" {
			envs[key] = value
		}
	}
}

func TestOpenTelemetry(t *testing.T) {
	ctx := t.Context()

	//initialize logger
	logger := logger.NewOpenTelemetry()
	if err := logger.Configure(envs); err != nil {
		t.Log(err)
		return
	}
	if err := logger.Open(ctx); err != nil {
		t.Log(err)
		return
	}
	defer logger.Close(ctx)

	//execute logging
	id := uuid.Must(uuid.NewRandom()).String()
	t.Log(id)

	logger.Error(ctx, "Error Hello, World!",
		slog.String("component", "testLog"),
		slog.String("id", id),
	)

	logger.Info(ctx, "Info Hello, World!",
		slog.String("component", "testLog"),
		slog.String("id", id),
	)

	logger.Debug(ctx, "Debug Hello, World!",
		slog.String("component", "testLog"),
		slog.String("id", id),
	)

	logger.Warn(ctx, "Warn Hello, World!",
		slog.String("component", "testLog"),
		slog.String("id", id),
	)

	<-time.After(10 * time.Second)
}

func TestSlog(t *testing.T) {
	ctx := t.Context()

	logger := logger.NewSlog()

	//initialize logger
	if err := logger.Configure(envs); err != nil {
		t.Log(err)
		return
	}
	if err := logger.Open(ctx); err != nil {
		t.Log(err)
		return
	}
	defer logger.Close(ctx)

	//execute logging
	id := uuid.Must(uuid.NewRandom()).String()
	t.Log(id)

	logger.Error(ctx, "Error Hello, World!",
		slog.String("component", "testLog"),
		slog.String("id", id),
	)

	logger.Info(ctx, "Info Hello, World!",
		slog.String("component", "testLog"),
		slog.String("id", id),
	)

	logger.Debug(ctx, "Debug Hello, World!",
		slog.String("component", "testLog"),
		slog.String("id", id),
	)

	logger.Warn(ctx, "Warn Hello, World!",
		slog.String("component", "testLog"),
		slog.String("id", id),
	)

	//log error
	logger.Error(ctx, "unable to perform operation",
		errors.ErrorCommon{
			ErrorMessage: "not implemeneted",
			ErrorType:    errors.ErrorTypeNotImplemented,
			DataId:       &id,
		})
}
