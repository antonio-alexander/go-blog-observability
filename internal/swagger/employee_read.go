package swagger

import (
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
)

// swagger:route GET /employees/{emp_no} Employee ReadEmployee
// Reads an employee using its id.
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
//   200: EmployeeGetResponseOk
//   401: EmployeeGetResponseUnauthorized
//   500: EmployeeGetResponseInternalServerError

// swagger:response EmployeeGetResponseOk
type EmployeeGetResponseOk struct {
	// in:body
	Employee data.Employee `json:"employee"`
}

// swagger:response EmployeeGetResponseUnauthorized
type EmployeeGetResponseUnauthorized struct {
	// in:body
	Body errors.ErrorUnauthorized
}

// swagger:response EmployeeGetResponseInternalServerError
type EmployeeGetResponseInternalServerError struct {
	// in:body
	Body errors.ErrorCommon
}

// swagger:parameters ReadEmployee
type EmployeeGetParams struct {
	// in:header
	CorrelationId string `json:"Correlation-Id"`

	// in:header
	// Authorization string `json:"Authorization"`

	// in:path
	EmpNo string `json:"emp_no"`
}
