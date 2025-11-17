package dto

import "time"

// APIResponse is the standard wrapper for all API responses
type APIResponse struct {
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorData  `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// ErrorData represents error information in API responses
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse creates a successful API response
func SuccessResponse(data interface{}) APIResponse {
	return APIResponse{
		Data:      data,
		Error:     nil,
		Timestamp: time.Now().UTC(),
	}
}

// ErrorResponse creates an error API response
func ErrorResponse(code, message string) APIResponse {
	return APIResponse{
		Data: nil,
		Error: &ErrorData{
			Code:    code,
			Message: message,
		},
		Timestamp: time.Now().UTC(),
	}
}

// ErrorResponseWithDetails creates an error API response with additional details
func ErrorResponseWithDetails(code, message, details string) APIResponse {
	return APIResponse{
		Data: nil,
		Error: &ErrorData{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC(),
	}
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalPages  int   `json:"total_pages"`
	TotalCount  int64 `json:"total_count"`
}

// PaginatedResponse wraps data with pagination information
type PaginatedResponse struct {
	Data       interface{}     `json:"data"`
	Pagination *PaginationMeta `json:"pagination"`
}
