package policy

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/logger"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/metrics"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/tracer"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/loader"
	"github.com/open-policy-agent/opa/v1/rego"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type policy struct {
	config struct {
		compileRegoFilename    string
		evaluationRegoFilename string
		regoVersion            string
	}
	tracer.Tracer
	logger.Logger
	metrics.Metrics
	regoVersion          ast.RegoVersion
	preparedQueryCompile struct {
		sync.RWMutex
		data map[string]*rego.PreparedEvalQuery
	}
	preparedQueryEvaluate struct {
		sync.RWMutex
		data map[string]*rego.PreparedEvalQuery
	}
	meter      metrics.Meter
	histograms struct {
		sync.RWMutex
		data map[string]metrics.Float64Histogram
	}
	opened atomic.Bool
}

func New(parameters ...any) interface {
	internal.Opener
	internal.Configurer
	Policy
} {
	p := &policy{regoVersion: ast.RegoV1}
	for _, parameter := range parameters {
		switch v := parameter.(type) {
		case tracer.Tracer:
			p.Tracer = v
		case metrics.Metrics:
			p.Metrics = v
		case logger.Logger:
			p.Logger = v
		}
	}
	return p
}

func (r *policy) createHistogram(histogramName string) (metrics.Float64Histogram, error) {
	r.histograms.Lock()
	defer r.histograms.Unlock()
	histogram, err := createHistogram(r.meter, histogramName)
	if err != nil {
		return nil, err
	}
	r.histograms.data[histogramName] = histogram
	return histogram, nil
}

func (r *policy) readPreparedQueryEvaluate(query string) *rego.PreparedEvalQuery {
	r.preparedQueryEvaluate.RLock()
	defer r.preparedQueryEvaluate.RUnlock()
	return r.preparedQueryEvaluate.data[query]
}

func (r *policy) writePreparedQueryEvaluate(query string, p *rego.PreparedEvalQuery) {
	r.preparedQueryEvaluate.Lock()
	defer r.preparedQueryEvaluate.Unlock()
	r.preparedQueryEvaluate.data[query] = p
}

func (r *policy) readPreparedQueryCompile(query string) *rego.PreparedEvalQuery {
	r.preparedQueryCompile.RLock()
	defer r.preparedQueryCompile.RUnlock()
	return r.preparedQueryCompile.data[query]
}

func (r *policy) writePreparedQueryCompile(query string, p *rego.PreparedEvalQuery) {
	r.preparedQueryCompile.Lock()
	defer r.preparedQueryCompile.Unlock()
	r.preparedQueryCompile.data[query] = p
}

func (r *policy) Configure(envs map[string]string) error {
	if s, ok := envs["REGO_COMPILE_FILENAME"]; ok && s != "" {
		r.config.compileRegoFilename = s
	}
	if s, ok := envs["EVAL_COMPILE_FILENAME"]; ok && s != "" {
		r.config.evaluationRegoFilename = s
	}
	if s, ok := envs["REGO_VERSION"]; ok && s != "" {
		r.config.regoVersion = s
	}
	return nil
}

func (r *policy) Open(ctx context.Context) error {
	if r.opened.Load() {
		return ErrAlreadyOpened
	}
	switch r.config.regoVersion {
	default:
		r.regoVersion = ast.DefaultRegoVersion
	case "v0":
		r.regoVersion = ast.RegoV0
	case "v1":
		r.regoVersion = ast.RegoV1
	}
	r.preparedQueryCompile.data = make(map[string]*rego.PreparedEvalQuery)
	r.preparedQueryEvaluate.data = make(map[string]*rego.PreparedEvalQuery)
	meter := r.Meter(packageName)
	r.meter = meter
	r.histograms.data = make(map[string]metrics.Float64Histogram)
	for _, histogramName := range histogramNames {
		if _, err := r.createHistogram(histogramName); err != nil {
			return err
		}
	}
	r.opened.Store(true)
	return nil
}

func (r *policy) Close(ctx context.Context) {
	if !r.opened.Load() {
		return
	}
	r.preparedQueryEvaluate.data = nil
	r.preparedQueryCompile.data = nil
	r.opened.Store(false)
}

func (r *policy) Compile(ctx context.Context, query string, input any, v ...any) (any, error) {
	ctx, span := r.Start(ctx, "policy.Compile", trace.WithAttributes(
		attribute.String("rego.query", query)))
	defer span.End()
	if !r.opened.Load() {
		return nil, ErrNotOpened
	}
	item, err := Evaluate(ctx,
		rego.Query(query),
		rego.Load([]string{r.config.compileRegoFilename}, nil),
		rego.SetRegoVersion(r.regoVersion),
		rego.Input(input),
	)
	if err != nil {
		return nil, ErrRego(err, query, r.config.compileRegoFilename)
	}
	if err := unmarshalItem(item, v...); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *policy) CompilePrepared(ctx context.Context, query string, input any, v ...any) (any, error) {
	ctx, span := r.Start(ctx, "policy.CompilePrepared", trace.WithAttributes(
		attribute.String("rego.query", query)))
	defer span.End()
	if !r.opened.Load() {
		return nil, ErrNotOpened
	}
	preparedQuery := r.readPreparedQueryCompile(query)
	if preparedQuery == nil {
		p, err := rego.New(
			rego.Query(query),
			rego.Load([]string{r.config.compileRegoFilename}, nil),
			rego.SetRegoVersion(r.regoVersion),
		).PrepareForEval(ctx)
		if err != nil {
			return nil, ErrRego(err, query, r.config.compileRegoFilename)
		}
		r.writePreparedQueryCompile(query, &p)
		preparedQuery = new(rego.PreparedEvalQuery)
		*preparedQuery = p
	}
	item, err := EvaluatePrepared(ctx, preparedQuery, rego.EvalInput(input))
	if err != nil {
		return nil, ErrRego(err, query, r.config.compileRegoFilename)
	}
	if err := unmarshalItem(item, v...); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *policy) Evaluate(ctx context.Context, query string, input any, v ...any) (any, error) {
	ctx, span := r.Start(ctx, "policy.Evaluate", trace.WithAttributes(
		attribute.String("rego.query", query)))
	defer span.End()
	if !r.opened.Load() {
		return nil, ErrNotOpened
	}
	item, err := Evaluate(ctx,
		rego.Query(query),
		rego.Load([]string{r.config.evaluationRegoFilename},
			loader.GlobExcludeName("compile*.rego", 1)),
		rego.SetRegoVersion(r.regoVersion),
		rego.Input(input),
	)
	if err != nil {
		return nil, ErrRego(err, query, r.config.evaluationRegoFilename)
	}
	if err := unmarshalItem(item, v...); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *policy) EvaluatePrepared(ctx context.Context, query string, input any, v ...any) (any, error) {
	ctx, span := r.Start(ctx, "policy.EvaluatePrepared", trace.WithAttributes(
		attribute.String("rego.query", query)))
	defer span.End()
	if !r.opened.Load() {
		return nil, ErrNotOpened
	}
	preparedQuery := r.readPreparedQueryEvaluate(query)
	if preparedQuery == nil {
		p, err := rego.New(
			rego.Query(query),
			rego.Load([]string{r.config.evaluationRegoFilename},
				loader.GlobExcludeName("compile*.rego", 1)),
			rego.SetRegoVersion(r.regoVersion),
		).PrepareForEval(ctx)
		if err != nil {
			return nil, ErrRego(err, query, r.config.evaluationRegoFilename)
		}
		r.writePreparedQueryEvaluate(query, &p)
		preparedQuery = new(rego.PreparedEvalQuery)
		*preparedQuery = p
	}
	item, err := EvaluatePrepared(ctx, preparedQuery, rego.EvalInput(input))
	if err != nil {
		return nil, ErrRego(err, query, r.config.evaluationRegoFilename)
	}
	if err := unmarshalItem(item, v...); err != nil {
		return nil, err
	}
	return item, nil
}
