package swagger

import (
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
)

// swagger:route POST /employees/{emp_no} Employee UpdateEmployee
// Updates an employee using its id.
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
//   200: EmployeePostResponseOk
//   401: EmployeePostResponseUnauthorized
//   500: EmployeePostResponseInternalServerError

// swagger:response EmployeePostResponseOk
type EmployeePostResponseOk struct {
	// in:body
	Employee data.Employee `json:"employee"`
}

// swagger:response EmployeePostResponseUnauthorized
type EmployeePostResponseUnauthorized struct {
	// in:body
	Body errors.ErrorUnauthorized
}

// swagger:response EmployeePostResponseInternalServerError
type EmployeePostResponseInternalServerError struct {
	// in:body
	Body errors.ErrorCommon
}

// swagger:parameters UpdateEmployee
type EmployeePostParams struct {
	// in:header
	CorrelationId string `json:"Correlation-Id"`

	// in:path
	EmpNo string `json:"emp_no"`

	// in:body
	EmployeePartial data.EmployeePartial `json:"employee_partial"`
}
