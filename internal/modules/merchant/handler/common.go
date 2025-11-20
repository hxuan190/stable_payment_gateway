package handler

import "time"

type APIResponse struct {
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorData  `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func SuccessResponse(data interface{}) APIResponse {
	return APIResponse{
		Data:      data,
		Error:     nil,
		Timestamp: time.Now().UTC(),
	}
}

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

type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalPages  int   `json:"total_pages"`
	TotalCount  int64 `json:"total_count"`
}

type PaginatedResponse struct {
	Data       interface{}     `json:"data"`
	Pagination *PaginationMeta `json:"pagination"`
}
