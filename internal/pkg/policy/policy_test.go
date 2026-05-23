package policy_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/policy"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/policy/tests"

	"github.com/stretchr/testify/assert"
)

var envs = map[string]string{
	//logging
	"LOG_LEVEL": "trace",
	//rego/policy
	"REGO_COMPILE_FILENAME": tests.RegoCompileFilename,
	"EVAL_COMPILE_FILENAME": tests.RegoEvaluationFilename,
}

func init() {
	envs := make(map[string]string)
	for _, env := range os.Environ() {
		if key, value, ok := strings.Cut(env, "="); ok && value != "" {
			envs[key] = value
		}
	}
}

type policyTest struct {
	policy interface {
		internal.Opener
		internal.Configurer
	}
	*tests.PolicyFixture
}

func newPolicyTest() *policyTest {
	policy := policy.New()
	return &policyTest{
		policy:        policy,
		PolicyFixture: tests.NewPolicyFixture(policy),
	}
}

func (p *policyTest) configure(envs map[string]string) error {
	if err := p.policy.Configure(envs); err != nil {
		return err
	}
	return nil
}

func (p *policyTest) open(ctx context.Context) error {
	if err := p.policy.Open(ctx); err != nil {
		return err
	}
	return nil
}

func (p *policyTest) close(ctx context.Context) {
	p.policy.Close(ctx)
}

func testPolicy(t *testing.T) {
	p := newPolicyTest()

	ctx := t.Context()
	err := p.configure(envs)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to configure test")
	}
	err = p.open(ctx)
	if !assert.Nil(t, err) {
		assert.FailNow(t, "unable to open test")
	}
	defer p.close(ctx)
	t.Run("Policy", p.TestPolicy(false))
	t.Run("Policy Prepared", p.TestPolicy(true))
}

func TestPolicy(t *testing.T) {
	testPolicy(t)
}
