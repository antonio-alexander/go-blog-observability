package swagger

// swagger:route POST /panic Panic Panic
// Panics.
//
// Consumes:
// - application/json
//
// Produces:
// - application/json
//
// responses:
//   500: PanicPostResponseOK

// swagger:response PanicPostResponseOK
type PanicPostResponseOK struct{}

// swagger:parameters Panic
type PanicPostParams struct {
	// in:header
	CorrelationId string `json:"Correlation-Id"`
}
