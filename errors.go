package camelmailer

import "fmt"

// APIError is a typed CamelMailer API failure. Every error envelope
// (and every non-2xx response) surfaces as *APIError, so callers can
// branch on the stable error code:
//
//	var apiErr *camelmailer.APIError
//	if errors.As(err, &apiErr) && apiErr.Code == "ValidationError" { … }
//
// Stable codes include Unauthorized, Forbidden, NotFound,
// ValidationError, ParameterMissing and InternalServerError.
type APIError struct {
	// Code is the stable machine-readable error code.
	Code string `json:"code"`
	// Message is the human-readable description.
	Message string `json:"message"`
	// StatusCode is the HTTP status of the response.
	StatusCode int `json:"-"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("camelmailer: %s: %s (HTTP %d)", e.Code, e.Message, e.StatusCode)
}
