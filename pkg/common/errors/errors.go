package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application error
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
	Cause   error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewAppError creates a new application error
func NewAppError(code, message string, status int, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  status,
		Cause:   cause,
	}
}

// Common error codes
const (
	CodeBadRequest         = "BAD_REQUEST"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeForbidden          = "FORBIDDEN"
	CodeNotFound           = "NOT_FOUND"
	CodeConflict           = "CONFLICT"
	CodeInternal           = "INTERNAL_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	CodeTimeout            = "TIMEOUT"
)

// Error constructors
func BadRequest(message string, cause error) *AppError {
	return NewAppError(CodeBadRequest, message, http.StatusBadRequest, cause)
}

func Unauthorized(message string, cause error) *AppError {
	return NewAppError(CodeUnauthorized, message, http.StatusUnauthorized, cause)
}

func Forbidden(message string, cause error) *AppError {
	return NewAppError(CodeForbidden, message, http.StatusForbidden, cause)
}

func NotFound(message string, cause error) *AppError {
	return NewAppError(CodeNotFound, message, http.StatusNotFound, cause)
}

func Conflict(message string, cause error) *AppError {
	return NewAppError(CodeConflict, message, http.StatusConflict, cause)
}

func Internal(message string, cause error) *AppError {
	return NewAppError(CodeInternal, message, http.StatusInternalServerError, cause)
}

func ServiceUnavailable(message string, cause error) *AppError {
	return NewAppError(CodeServiceUnavailable, message, http.StatusServiceUnavailable, cause)
}

func Timeout(message string, cause error) *AppError {
	return NewAppError(CodeTimeout, message, http.StatusGatewayTimeout, cause)
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// FromError converts error to AppError
func FromError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return Internal("An unexpected error occurred", err)
}
