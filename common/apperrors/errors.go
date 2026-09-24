package apperrors

import (
	"errors"
	"net/http"
)

// Sentinel errors services return; handlers map these to HTTP statuses.
// Wrap underlying errors with fmt.Errorf("...: %w", ErrNotFound) so
// callers can still errors.Is() against these.
var (
	ErrNotFound     = errors.New("resource not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrValidation   = errors.New("validation failed")
	ErrConflict     = errors.New("resource conflict")
)

// ToHTTPStatus maps a sentinel error to the HTTP status code a handler
// should return. Unknown errors default to 500 — never leak internal
// error details for those.
func ToHTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
