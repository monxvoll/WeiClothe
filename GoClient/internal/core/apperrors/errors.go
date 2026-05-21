package apperrors

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidID          = errors.New("invalid id")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
)

// ValidationError carries a client-safe field hint while wrapping ErrInvalidInput.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "invalid " + e.Field
}

func (e *ValidationError) Unwrap() error {
	return ErrInvalidInput
}
