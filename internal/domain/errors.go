// Package domain holds pure entities, value objects, enums and typed errors.
// It imports nothing from the rest of the module and performs no I/O.
package domain

import "errors"

// Typed errors that the service layer returns. One place (the HTTP layer) maps
// them to status codes and user-facing messages; callers compare with
// errors.Is and wrap with %w.
var (
	// ErrNotFound means the requested resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict means the request conflicts with current state.
	ErrConflict = errors.New("conflict")
	// ErrValidation means the request failed validation.
	ErrValidation = errors.New("validation failed")
	// ErrRateLimited means the caller exceeded a rate limit.
	ErrRateLimited = errors.New("rate limited")
	// ErrForbidden means the caller is not allowed to perform the action.
	ErrForbidden = errors.New("forbidden")
)

// ValidationError carries a field-level message with the shared ErrValidation
// sentinel so handlers can render a helpful response without leaking internals.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Field == "" {
		return ErrValidation.Error() + ": " + e.Message
	}
	return ErrValidation.Error() + ": " + e.Field + ": " + e.Message
}

// Is reports that a ValidationError matches the ErrValidation sentinel.
func (e *ValidationError) Is(target error) bool { return target == ErrValidation }

// Unwrap exposes the sentinel for errors.Is.
func (e *ValidationError) Unwrap() error { return ErrValidation }
