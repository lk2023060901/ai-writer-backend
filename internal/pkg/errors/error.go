package errors

import (
	"errors"
	"fmt"
)

// AppError represents a structured application error
type AppError struct {
	Code    int    // Business error code
	Message string // Human-readable message
	Err     error  // Underlying error (if any)
	Details string // Additional details
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	if e.Details != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap implements error unwrapping for errors.Is and errors.As
func (e *AppError) Unwrap() error {
	return e.Err
}

// HTTPStatus returns the HTTP status code for this error
func (e *AppError) HTTPStatus() int {
	return GetHTTPStatus(e.Code)
}

// New creates a new AppError with the given code
func New(code int, details ...string) *AppError {
	detail := ""
	if len(details) > 0 {
		detail = details[0]
	}
	return &AppError{
		Code:    code,
		Message: GetMessage(code),
		Details: detail,
	}
}

// Wrap wraps an existing error with an error code
func Wrap(err error, code int, details ...string) *AppError {
	if err == nil {
		return nil
	}

	// If already an AppError, update details if provided
	var appErr *AppError
	if errors.As(err, &appErr) {
		if len(details) > 0 && details[0] != "" {
			appErr.Details = details[0]
		}
		return appErr
	}

	detail := ""
	if len(details) > 0 {
		detail = details[0]
	}

	return &AppError{
		Code:    code,
		Message: GetMessage(code),
		Err:     err,
		Details: detail,
	}
}

// Wrapf wraps an error with formatted details
func Wrapf(err error, code int, format string, args ...interface{}) *AppError {
	return Wrap(err, code, fmt.Sprintf(format, args...))
}

// Is checks if err is an AppError with the given code
func Is(err error, code int) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// ExtractCode extracts the error code from an error
func ExtractCode(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ErrInternalServer
}

// GetDetails extracts error details
func GetDetails(err error) string {
	var appErr *AppError
	if errors.As(err, &appErr) {
		if appErr.Details != "" {
			return appErr.Details
		}
		if appErr.Err != nil {
			return appErr.Err.Error()
		}
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

// Common error constructors for convenience

// NewInternalError creates an internal server error
func NewInternalError(details ...string) *AppError {
	return New(ErrInternalServer, details...)
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *AppError {
	return New(ErrNotFound, resource)
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(details ...string) *AppError {
	return New(ErrUnauthorized, details...)
}

// NewForbiddenError creates a forbidden error
func NewForbiddenError(details ...string) *AppError {
	return New(ErrForbidden, details...)
}

// NewBadRequestError creates a bad request error
func NewBadRequestError(details ...string) *AppError {
	return New(ErrBadRequest, details...)
}

// NewConflictError creates a conflict error
func NewConflictError(resource string) *AppError {
	return New(ErrConflict, resource)
}

// NewValidationError creates a validation error
func NewValidationError(field string) *AppError {
	return New(ErrInvalidParams, fmt.Sprintf("validation failed for field: %s", field))
}
