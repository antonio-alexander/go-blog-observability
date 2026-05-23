package policy

import (
	"context"
	"encoding/json"

	"github.com/open-policy-agent/opa/v1/rego"
)

func unmarshalItem(item any, v ...any) error {
	if len(v) <= 0 {
		return nil
	}
	bytes, err := json.Marshal(item)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bytes, v[0]); err != nil {
		return ErrJson(err)
	}
	return nil
}

func Evaluate(ctx context.Context, options ...func(r *rego.Rego)) (any, error) {
	results, err := rego.New(options...).Eval(ctx)
	if err != nil {
		return nil, err
	}
	if len(results) <= 0 || len(results[0].Expressions) <= 0 {
		return nil, ErrRegoResultsEmpty
	}
	return results[0].Expressions[0].Value, nil
}

func EvaluatePrepared(ctx context.Context, preparedQuery *rego.PreparedEvalQuery,
	options ...rego.EvalOption) (any, error) {
	results, err := preparedQuery.Eval(ctx, options...)
	if err != nil {
		return nil, err
	}
	if len(results) <= 0 || len(results[0].Expressions) <= 0 {
		return nil, ErrRegoResultsEmpty
	}
	return results[0].Expressions[0].Value, nil
}
