package swagger

import "github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"

// swagger:route DELETE /employees/{emp_no} Employee DeleteEmployee
// Deletes an employee using its id.
//
// Consumes:
// - application/json
//
// Produces:
// - application/json
//
// Security:
//   BearerAuth:
//
// responses:
//   200: EmployeeDeleteResponseOk
//   401: EmployeeDeleteResponseUnauthorized
//   500: EmployeeDeleteResponseInternalServerError

// swagger:response EmployeeDeleteResponseOk
type EmployeeDeleteResponseOk struct{}

// swagger:response EmployeeDeleteResponseUnauthorized
type EmployeeDeleteResponseUnauthorized struct {
	// in:body
	Body errors.ErrorUnauthorized
}

// swagger:response EmployeeDeleteResponseInternalServerError
type EmployeeDeleteResponseInternalServerError struct {
	// in:body
	Body errors.ErrorCommon
}

// swagger:parameters DeleteEmployee
type EmployeeDeleteParams struct {
	// in:header
	CorrelationId string `json:"Correlation-Id"`

	// in:path
	EmpNo string `json:"emp_no"`
}
