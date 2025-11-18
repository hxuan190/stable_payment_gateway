package dto

import (
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
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

// KYCDocumentToResponse converts a model.KYCDocument to UploadKYCDocumentResponse
func KYCDocumentToResponse(doc *model.KYCDocument) UploadKYCDocumentResponse {
	fileSizeKB := float64(doc.FileSizeBytes) / 1024.0
	return UploadKYCDocumentResponse{
		DocumentID:   doc.ID.String(),
		MerchantID:   doc.MerchantID.String(),
		DocumentType: string(doc.DocumentType),
		FileURL:      doc.FileURL,
		FileSizeKB:   fileSizeKB,
		MimeType:     doc.MimeType,
		Status:       string(doc.Status),
		CreatedAt:    doc.CreatedAt,
	}
}

// KYCDocumentToListItem converts a model.KYCDocument to KYCDocumentItem
func KYCDocumentToListItem(doc *model.KYCDocument) KYCDocumentItem {
	fileSizeKB := float64(doc.FileSizeBytes) / 1024.0
	item := KYCDocumentItem{
		DocumentID:   doc.ID.String(),
		DocumentType: string(doc.DocumentType),
		FileURL:      doc.FileURL,
		FileSizeKB:   fileSizeKB,
		MimeType:     doc.MimeType,
		Status:       string(doc.Status),
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
	}

	// Handle optional fields
	if doc.ReviewedBy.Valid {
		reviewedBy := doc.ReviewedBy.UUID.String()
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
