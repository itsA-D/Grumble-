package web

import (
	"errors"
	"net/http"

	"github.com/OWNER/painboard/internal/domain"
)

// errorStatus maps a domain error to an HTTP status. It is the one place that
// translates service errors into status codes (CLAUDE.md error conventions);
// internals are never leaked to the client.
func errorStatus(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrRateLimited):
		return http.StatusTooManyRequests
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, domain.ErrValidation):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// errorMessage is the user-facing sentence for a domain error.
func errorMessage(err error) string {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return "We could not find that page."
	case errors.Is(err, domain.ErrRateLimited):
		return "You are doing that too often. Please wait a moment and try again."
	case errors.Is(err, domain.ErrForbidden):
		return "We could not verify that request. Please reload the page and try again."
	case errors.Is(err, domain.ErrConflict):
		return "That action conflicts with the current state."
	case errors.Is(err, domain.ErrValidation):
		return "Please check the form and try again."
	default:
		return "Something went wrong. Please try again."
	}
}
