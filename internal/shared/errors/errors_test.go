package errors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	err := New(ErrCodeNotFound, "User not found", http.StatusNotFound)

	assert.Equal(t, ErrCodeNotFound, err.Code)
	assert.Equal(t, "User not found", err.Message)
	assert.Equal(t, http.StatusNotFound, err.StatusCode)
}

func TestAppError_Error(t *testing.T) {
	err := New(ErrCodeValidation, "Invalid input", http.StatusBadRequest)
	assert.Equal(t, "VALIDATION_ERROR: Invalid input", err.Error())

	wrapped := Wrap(errors.New("db error"), ErrCodeInternal, "Database error", http.StatusInternalServerError)
	assert.Contains(t, wrapped.Error(), "INTERNAL_ERROR")
	assert.Contains(t, wrapped.Error(), "Database error")
	assert.Contains(t, wrapped.Error(), "db error")
}

func TestAppError_WithDetails(t *testing.T) {
	err := Validation("Invalid email").
		WithDetails("field", "email").
		WithDetails("reason", "format")

	assert.Equal(t, "email", err.Details["field"])
	assert.Equal(t, "format", err.Details["reason"])
}

func TestNotFound(t *testing.T) {
	err := NotFound("Payment")

	assert.Equal(t, ErrCodeNotFound, err.Code)
	assert.Equal(t, "Payment not found", err.Message)
	assert.Equal(t, http.StatusNotFound, err.StatusCode)
}

func TestValidation(t *testing.T) {
	err := Validation("Invalid amount")

	assert.Equal(t, ErrCodeValidation, err.Code)
	assert.Equal(t, "Invalid amount", err.Message)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
}

func TestIsAppError(t *testing.T) {
	appErr := NotFound("Payment")
	stdErr := errors.New("standard error")

	assert.True(t, IsAppError(appErr))
	assert.False(t, IsAppError(stdErr))
}

func TestGetAppError(t *testing.T) {
	appErr := NotFound("Payment")
	stdErr := errors.New("standard error")

	extracted := GetAppError(appErr)
	assert.NotNil(t, extracted)
	assert.Equal(t, ErrCodeNotFound, extracted.Code)

	extracted = GetAppError(stdErr)
	assert.Nil(t, extracted)
}

func TestGetStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "AppError returns correct status",
			err:      NotFound("Payment"),
			expected: http.StatusNotFound,
		},
		{
			name:     "Standard error returns 500",
			err:      errors.New("standard error"),
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := GetStatusCode(tt.err)
			assert.Equal(t, tt.expected, status)
		})
	}
}

func TestInsufficientBalance(t *testing.T) {
	err := InsufficientBalance("")
	assert.Equal(t, "Insufficient balance", err.Message)

	err = InsufficientBalance("Custom message")
	assert.Equal(t, "Custom message", err.Message)
}
