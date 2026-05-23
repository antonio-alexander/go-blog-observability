package policy

import (
	"context"

	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
)

const packageName string = "github.com/antonio-alexander/go-blog-observability/internal/pkg/policy"

var (
	ErrRegoResultsEmpty = errors.Must(errors.New("rego results are empty"))
	ErrNotOpened        = errors.Must(errors.New("not opened"))
	ErrAlreadyOpened    = errors.Must(errors.New("already opened"))
)

type Policy interface {
	Compile(ctx context.Context, query string, input any, v ...any) (any, error)
	CompilePrepared(ctx context.Context, query string, input any, v ...any) (any, error)
	Evaluate(ctx context.Context, query string, input any, v ...any) (any, error)
	EvaluatePrepared(ctx context.Context, query string, input any, v ...any) (any, error)
}
