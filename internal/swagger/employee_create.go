package swagger

import (
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
)

// swagger:route PUT /employees Employee CreateEmployee
// Creates an employee.
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
//   200: EmployeePutResponseOk
//   401: EmployeePutResponseUnauthorized
//   500: EmployeePutResponseInternalServerError

// swagger:response EmployeePutResponseOk
type EmployeePutResponseOk struct {
	// in:body
	Employee data.Employee `json:"employee"`
}

// swagger:response EmployeePutResponseUnauthorized
type EmployeePutResponseUnauthorized struct {
	// in:body
	Body errors.ErrorUnauthorized
}

// swagger:response EmployeePutResponseInternalServerError
type EmployeePutResponseInternalServerError struct {
	// in:body
	Body errors.ErrorCommon
}

// swagger:parameters CreateEmployee
type EmployeePutParams struct {
	// in:header
	CorrelationId string `json:"Correlation-Id"`

	// in:body
	EmployeePartial data.EmployeePartial `json:"employee_partial"`
}
