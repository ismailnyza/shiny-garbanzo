package errors

import "fmt"

// AppError represents a domain-level error with a code and message.
type AppError struct {
	Code    string
	Message string
	Details interface{}
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

const (
	CodeNotFound       = "NOT_FOUND"
	CodeConflict       = "CONFLICT"
	CodeForbidden      = "FORBIDDEN"
	CodeUnauthorized   = "UNAUTHORIZED"
	CodeValidation     = "VALIDATION_ERROR"
	CodeInternal       = "INTERNAL_ERROR"
	CodeUnprocessable  = "UNPROCESSABLE"
	CodeFileTooLarge   = "FILE_TOO_LARGE"
)

func NewNotFound(message string) *AppError {
	return &AppError{Code: CodeNotFound, Message: message}
}

func NewConflict(message string) *AppError {
	return &AppError{Code: CodeConflict, Message: message}
}

func NewForbidden(message string) *AppError {
	return &AppError{Code: CodeForbidden, Message: message}
}

func NewUnauthorized(message string) *AppError {
	return &AppError{Code: CodeUnauthorized, Message: message}
}

func NewValidation(message string, details interface{}) *AppError {
	return &AppError{Code: CodeValidation, Message: message, Details: details}
}

func NewInternal(err error) *AppError {
	return &AppError{Code: CodeInternal, Message: "An unexpected error occurred", Err: err}
}

func NewUnprocessable(message string) *AppError {
	return &AppError{Code: CodeUnprocessable, Message: message}
}

func NewFileTooLarge(message string) *AppError {
	return &AppError{Code: CodeFileTooLarge, Message: message}
}
