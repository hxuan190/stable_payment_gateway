package types

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

// Offset calculates the database offset
func (p PaginationParams) Offset() int {
	if p.Page <= 0 {
		return 0
	}
	return (p.Page - 1) * p.GetPageSize()
}

// GetPageSize returns the page size with defaults
func (p PaginationParams) GetPageSize() int {
	if p.PageSize <= 0 {
		return 20 // Default page size
	}
	if p.PageSize > 100 {
		return 100 // Max page size
	}
	return p.PageSize
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data       interface{}      `json:"data"`
	Pagination PaginationMeta   `json:"pagination"`
}

// PaginationMeta contains pagination metadata
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// NewPaginationMeta creates pagination metadata
func NewPaginationMeta(params PaginationParams, totalItems int64) PaginationMeta {
	pageSize := params.GetPageSize()
	totalPages := int((totalItems + int64(pageSize) - 1) / int64(pageSize))
	page := params.Page
	if page <= 0 {
		page = 1
	}

	return PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}
