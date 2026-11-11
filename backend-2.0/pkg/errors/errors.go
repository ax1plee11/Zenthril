package errors

import "fmt"

type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func New(code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(err error, code, message string) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

const (
	ErrCodeInvalidInput     = "INVALID_INPUT"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeForbidden        = "FORBIDDEN"
	ErrCodeInternal         = "INTERNAL_ERROR"
	ErrCodeConflict         = "CONFLICT"
	ErrCodeTooManyRequests  = "TOO_MANY_REQUESTS"
	ErrCodeUnavailable      = "SERVICE_UNAVAILABLE"
)

var ErrNotFound = &AppError{Code: "NOT_FOUND", Message: "resource not found"}

func IsNotFound(err error) bool {
	return err != nil && err == ErrNotFound
}
