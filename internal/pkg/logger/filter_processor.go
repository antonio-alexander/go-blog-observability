package logger

import (
	"context"
	"log/slog"

	log "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type filterProcessor struct {
	level slog.Level
	*sdklog.BatchProcessor
}

func newFilterProcessor(s *sdklog.BatchProcessor, level slog.Level) sdklog.Processor {
	return &filterProcessor{
		BatchProcessor: s,
		level:          level,
	}
}

func (f *filterProcessor) OnEmit(ctx context.Context, record *sdklog.Record) error {
	if record.Severity() < log.Severity(f.level) {
		return nil
	}
	return f.BatchProcessor.OnEmit(ctx, record)
}
