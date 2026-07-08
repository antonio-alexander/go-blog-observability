package swagger

import (
	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
)

// swagger:route POST /authorize Authorize Authorize
// Generates a signed authz token.
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
//   200: AuthorizePostResponseOk
//   500: AuthorizePostResponseInternalServerError

// swagger:response AuthorizePostResponseOk
type AuthorizePostResponseOk struct {
	// in:body
	Authorization string `json:"authorization"`
}

// swagger:response AuthorizePostResponseInternalServerError
type AuthorizePostResponseInternalServerError struct {
	// in:body
	Body errors.ErrorCommon
}

// swagger:parameters Authorize
type AuthorizePostParams struct {
	// in:header
	CorrelationId string `json:"Correlation-Id"`

	// in:body
	Body struct {
		UserId string `json:"user_id"`
	}
}
