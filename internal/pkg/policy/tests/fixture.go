package tests

import (
	"testing"

	"github.com/antonio-alexander/go-blog-observability/internal"
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/policy"

	"github.com/stretchr/testify/assert"
)

type PolicyFixture struct {
	policy.Policy
}

func NewPolicyFixture(parameters ...any) *PolicyFixture {
	p := &PolicyFixture{}
	for _, parameter := range parameters {
		switch parameter := parameter.(type) {
		case policy.Policy:
			p.Policy = parameter
		}
	}
	return p
}

func (p *PolicyFixture) TestPolicy(prepared bool) func(t *testing.T) {
	return func(t *testing.T) {
		var err error
		var item any

		//generate dynamic constants
		userId := internal.GenerateId()

		// generate context
		ctx := t.Context()

		//compile
		compileInput := data.PolicyCompileInput{
			Roles: map[string]data.PolicyRole{
				"employee_admin": {
					Name:    "employee_admin",
					UserIds: []string{userId},
					Permissions: []data.PolicyPermission{
						data.PolicyPermissionEmployeeCreate,
						data.PolicyPermissionEmployeeRead,
						data.PolicyPermissionEmployeeUpdate,
						data.PolicyPermissionEmployeeDelete,
					},
				},
			},
		}
		compileOutput := &data.PolicyCompileOutput{}
		if !prepared {
			item, err = p.Compile(ctx, "data.opa.compilation.resources", compileInput, compileOutput)
		} else {
			item, err = p.CompilePrepared(ctx, "data.opa.compilation.resources", compileInput, compileOutput)
		}
		assert.Nil(t, err)
		assert.NotNil(t, item)

		//validate compilation
		if assert.NotNil(t, compileOutput.Access) {
			if assert.NotNil(t, compileOutput.Access.Employees) {
				if assert.NotEmpty(t, compileOutput.Access.Employees.Create) {
					assert.Contains(t, compileOutput.Access.Employees.Create, userId)
				}
				if assert.NotEmpty(t, compileOutput.Access.Employees.Read) {
					assert.Contains(t, compileOutput.Access.Employees.Read, userId)
				}
				if assert.NotEmpty(t, compileOutput.Access.Employees.Update) {
					assert.Contains(t, compileOutput.Access.Employees.Update, userId)
				}
				if assert.NotEmpty(t, compileOutput.Access.Employees.Delete) {
					assert.Contains(t, compileOutput.Access.Employees.Delete, userId)
				}
			}
		}

		//evaluate
		evaluateInput := data.PolicyEvaluationInput{
			TokenUserId:         userId,
			PolicyCompileOutput: compileOutput,
		}
		evaluateOutput := &data.PolicyEvaluateOutput{}
		if !prepared {
			item, err = p.Evaluate(ctx, "data.opa.evaluation", evaluateInput, evaluateOutput)
		} else {
			item, err = p.EvaluatePrepared(ctx, "data.opa.evaluation", evaluateInput, evaluateOutput)
		}
		assert.Nil(t, err)
		assert.NotNil(t, item)

		//validate evaluation
		assert.True(t, evaluateOutput.CanCreateEmployee)
		assert.True(t, evaluateOutput.CanReadEmployee)
		assert.True(t, evaluateOutput.CanUpdateEmployee)
		assert.True(t, evaluateOutput.CanDeleteEmployee)
	}
}
