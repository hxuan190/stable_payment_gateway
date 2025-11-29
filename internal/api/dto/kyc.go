package dto

import (
	"time"

	merchantDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/domain"
)

// UploadKYCDocumentRequest represents the request to upload a KYC document
// Note: File is uploaded via multipart/form-data
type UploadKYCDocumentRequest struct {
	DocumentType string `form:"document_type" binding:"required" validate:"required,oneof=business_registration tax_certificate bank_statement id_card utility_bill"`
	// File is handled separately via c.FormFile("file")
}

// UploadKYCDocumentResponse represents the response after uploading a KYC document
type UploadKYCDocumentResponse struct {
	DocumentID   string    `json:"document_id"`
	MerchantID   string    `json:"merchant_id"`
	DocumentType string    `json:"document_type"`
	FileURL      string    `json:"file_url"`
	FileSizeKB   float64   `json:"file_size_kb"`
	MimeType     string    `json:"mime_type"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// ListKYCDocumentsResponse represents the response when listing KYC documents
type ListKYCDocumentsResponse struct {
	Documents []KYCDocumentItem `json:"documents"`
}

// KYCDocumentItem represents a KYC document in a list
type KYCDocumentItem struct {
	DocumentID    string     `json:"document_id"`
	DocumentType  string     `json:"document_type"`
	FileURL       string     `json:"file_url"`
	FileSizeKB    float64    `json:"file_size_kb"`
	MimeType      string     `json:"mime_type"`
	Status        string     `json:"status"`
	ReviewedBy    *string    `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	ReviewerNotes *string    `json:"reviewer_notes,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// KYCDocumentToResponse converts a merchantDomain.KYCDocument to UploadKYCDocumentResponse
func KYCDocumentToResponse(doc *merchantDomain.KYCDocument) UploadKYCDocumentResponse {
	// Handle nullable FileSizeBytes
	var fileSizeKB float64
	if doc.FileSizeBytes.Valid {
		fileSizeKB = float64(doc.FileSizeBytes.Int64) / 1024.0
	}

	// Handle nullable MimeType
	mimeType := ""
	if doc.MimeType.Valid {
		mimeType = doc.MimeType.String
	}

	return UploadKYCDocumentResponse{
		DocumentID:   doc.ID,         // ID is already a string
		MerchantID:   doc.MerchantID, // MerchantID is already a string
		DocumentType: string(doc.DocumentType),
		FileURL:      doc.FileURL,
		FileSizeKB:   fileSizeKB,
		MimeType:     mimeType,
		Status:       string(doc.Status),
		CreatedAt:    doc.CreatedAt,
	}
}

// KYCDocumentToListItem converts a merchantDomain.KYCDocument to KYCDocumentItem
func KYCDocumentToListItem(doc *merchantDomain.KYCDocument) KYCDocumentItem {
	// Handle nullable FileSizeBytes
	var fileSizeKB float64
	if doc.FileSizeBytes.Valid {
		fileSizeKB = float64(doc.FileSizeBytes.Int64) / 1024.0
	}

	// Handle nullable MimeType
	mimeType := ""
	if doc.MimeType.Valid {
		mimeType = doc.MimeType.String
	}

	item := KYCDocumentItem{
		DocumentID:   doc.ID, // ID is already a string
		DocumentType: string(doc.DocumentType),
		FileURL:      doc.FileURL,
		FileSizeKB:   fileSizeKB,
		MimeType:     mimeType,
		Status:       string(doc.Status),
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
	}

	// Handle optional fields
	if doc.ReviewedBy.Valid {
		reviewedBy := doc.ReviewedBy.String // ReviewedBy is sql.NullString, not UUID
		item.ReviewedBy = &reviewedBy
	}
	if doc.ReviewedAt.Valid {
		reviewedAt := doc.ReviewedAt.Time
		item.ReviewedAt = &reviewedAt
	}
	if doc.ReviewerNotes.Valid {
		reviewerNotes := doc.ReviewerNotes.String
		item.ReviewerNotes = &reviewerNotes
	}

	return item
}
