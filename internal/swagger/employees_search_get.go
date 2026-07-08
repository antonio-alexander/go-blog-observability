package swagger

import (
	"github.com/antonio-alexander/go-blog-observability/internal/data"
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
)

// swagger:route GET /employees/search Employee SearchEmployee
// Searches employees using search criteria.
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
//   200: EmployeeSearchResponseOk
//   401: EmployeeSearchResponseUnauthorized
//   500: EmployeeSearchResponseInternalServerError

// swagger:response EmployeeSearchResponseOk
type EmployeeSearchGetResponseOk struct {
	// in:body
	Employees []data.Employee `json:"employees"`
}

// swagger:response EmployeeSearchResponseUnauthorized
type EmployeeSearchResponseUnauthorized struct {
	// in:body
	Body errors.ErrorUnauthorized
}

// swagger:response EmployeeSearchResponseInternalServerError
type EmployeeSearchResponseInternalServerError struct {
	// in:body
	Body errors.ErrorCommon
}

// swagger:parameters SearchEmployee
type EmployeeSearchGetParams struct {
	// in:header
	CorrelationId string `json:"Correlation-Id"`

	// in:query
	data.EmployeeSearch
}
