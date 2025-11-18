package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode represents a standard error code
type ErrorCode string

const (
	// Common errors
	ErrCodeInternal         ErrorCode = "INTERNAL_ERROR"
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeValidation       ErrorCode = "VALIDATION_ERROR"
	ErrCodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden        ErrorCode = "FORBIDDEN"
	ErrCodeConflict         ErrorCode = "CONFLICT"
	ErrCodeRateLimited      ErrorCode = "RATE_LIMITED"
	ErrCodeBadRequest       ErrorCode = "BAD_REQUEST"

	// Business errors
	ErrCodeInsufficientBalance ErrorCode = "INSUFFICIENT_BALANCE"
	ErrCodeInvalidStatus       ErrorCode = "INVALID_STATUS"
	ErrCodeDuplicate           ErrorCode = "DUPLICATE"
	ErrCodeExpired             ErrorCode = "EXPIRED"
	ErrCodeLimitExceeded       ErrorCode = "LIMIT_EXCEEDED"
)

// AppError represents an application error with code and message
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	StatusCode int                    `json:"-"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Err        error                  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap implements error unwrapping
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithError wraps another error
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// New creates a new AppError
func New(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Wrap wraps an existing error with an AppError
func Wrap(err error, code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
	}
}

// Common error constructors

// NotFound creates a NOT_FOUND error
func NotFound(resource string) *AppError {
	return New(
		ErrCodeNotFound,
		fmt.Sprintf("%s not found", resource),
		http.StatusNotFound,
	)
}

// Validation creates a VALIDATION_ERROR
func Validation(message string) *AppError {
	return New(
		ErrCodeValidation,
		message,
		http.StatusBadRequest,
	)
}

// Unauthorized creates an UNAUTHORIZED error
func Unauthorized(message string) *AppError {
	if message == "" {
		message = "Unauthorized"
	}
	return New(
		ErrCodeUnauthorized,
		message,
		http.StatusUnauthorized,
	)
}

// Forbidden creates a FORBIDDEN error
func Forbidden(message string) *AppError {
	if message == "" {
		message = "Forbidden"
	}
	return New(
		ErrCodeForbidden,
		message,
		http.StatusForbidden,
	)
}

// Conflict creates a CONFLICT error
func Conflict(message string) *AppError {
	return New(
		ErrCodeConflict,
		message,
		http.StatusConflict,
	)
}

// Internal creates an INTERNAL_ERROR
func Internal(message string) *AppError {
	if message == "" {
		message = "Internal server error"
	}
	return New(
		ErrCodeInternal,
		message,
		http.StatusInternalServerError,
	)
}

// BadRequest creates a BAD_REQUEST error
func BadRequest(message string) *AppError {
	return New(
		ErrCodeBadRequest,
		message,
		http.StatusBadRequest,
	)
}

// InsufficientBalance creates an INSUFFICIENT_BALANCE error
func InsufficientBalance(message string) *AppError {
	if message == "" {
		message = "Insufficient balance"
	}
	return New(
		ErrCodeInsufficientBalance,
		message,
		http.StatusBadRequest,
	)
}

// InvalidStatus creates an INVALID_STATUS error
func InvalidStatus(message string) *AppError {
	return New(
		ErrCodeInvalidStatus,
		message,
		http.StatusBadRequest,
	)
}

// Expired creates an EXPIRED error
func Expired(message string) *AppError {
	return New(
		ErrCodeExpired,
		message,
		http.StatusBadRequest,
	)
}

// LimitExceeded creates a LIMIT_EXCEEDED error
func LimitExceeded(message string) *AppError {
	return New(
		ErrCodeLimitExceeded,
		message,
		http.StatusBadRequest,
	)
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// GetAppError extracts AppError from error
func GetAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// GetStatusCode returns HTTP status code for an error
func GetStatusCode(err error) int {
	if appErr := GetAppError(err); appErr != nil {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}
