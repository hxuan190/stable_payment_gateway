package domain

import (
	"database/sql"
	"time"
)

// KYCDocumentStatus represents the review status of a KYC document
type KYCDocumentStatus string

const (
	KYCDocumentStatusPending  KYCDocumentStatus = "pending"
	KYCDocumentStatusApproved KYCDocumentStatus = "approved"
	KYCDocumentStatusRejected KYCDocumentStatus = "rejected"
)

// KYCDocumentType represents the type of KYC document
type KYCDocumentType string

const (
	KYCDocumentTypeBusinessRegistration KYCDocumentType = "business_registration"
	KYCDocumentTypeTaxCertificate       KYCDocumentType = "tax_certificate"
	KYCDocumentTypeOwnerID              KYCDocumentType = "owner_id"
	KYCDocumentTypeBankStatement        KYCDocumentType = "bank_statement"
	KYCDocumentTypeOther                KYCDocumentType = "other"
)

// KYCDocument represents a KYC document uploaded by a merchant
type KYCDocument struct {
	ID         string `json:"id" db:"id"`
	MerchantID string `json:"merchant_id" db:"merchant_id" validate:"required,uuid"`

	// Document information
	DocumentType  string         `json:"document_type" db:"document_type" validate:"required,max=50"`
	FileURL       string         `json:"file_url" db:"file_url" validate:"required,url"`
	FileSizeBytes sql.NullInt64  `json:"file_size_bytes,omitempty" db:"file_size_bytes"`
	MimeType      sql.NullString `json:"mime_type,omitempty" db:"mime_type" validate:"omitempty,max=100"`

	// Review status
	Status        KYCDocumentStatus `json:"status" db:"status" validate:"required,oneof=pending approved rejected"`
	ReviewedBy    sql.NullString    `json:"reviewed_by,omitempty" db:"reviewed_by" validate:"omitempty,uuid"`
	ReviewedAt    sql.NullTime      `json:"reviewed_at,omitempty" db:"reviewed_at"`
	ReviewerNotes sql.NullString    `json:"reviewer_notes,omitempty" db:"reviewer_notes"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// IsPending returns true if the document is pending review
func (d *KYCDocument) IsPending() bool {
	return d.Status == KYCDocumentStatusPending
}

// IsApproved returns true if the document has been approved
func (d *KYCDocument) IsApproved() bool {
	return d.Status == KYCDocumentStatusApproved
}

// IsRejected returns true if the document has been rejected
func (d *KYCDocument) IsRejected() bool {
	return d.Status == KYCDocumentStatusRejected
}

// HasBeenReviewed returns true if the document has been reviewed
func (d *KYCDocument) HasBeenReviewed() bool {
	return d.ReviewedAt.Valid
}

// GetReviewerNotes returns the reviewer notes if available
func (d *KYCDocument) GetReviewerNotes() string {
	if d.ReviewerNotes.Valid {
		return d.ReviewerNotes.String
	}
	return ""
}

// GetFileSizeBytes returns the file size if available
func (d *KYCDocument) GetFileSizeBytes() int64 {
	if d.FileSizeBytes.Valid {
		return d.FileSizeBytes.Int64
	}
	return 0
}

// GetMimeType returns the MIME type if available
func (d *KYCDocument) GetMimeType() string {
	if d.MimeType.Valid {
		return d.MimeType.String
	}
	return ""
}

// IsValidDocumentType checks if the document type is valid
func IsValidDocumentType(docType string) bool {
	validTypes := map[string]bool{
		string(KYCDocumentTypeBusinessRegistration): true,
		string(KYCDocumentTypeTaxCertificate):       true,
		string(KYCDocumentTypeOwnerID):              true,
		string(KYCDocumentTypeBankStatement):        true,
		string(KYCDocumentTypeOther):                true,
	}
	return validTypes[docType]
}

// IsValidMimeType checks if the MIME type is allowed for KYC documents
func IsValidMimeType(mimeType string) bool {
	allowedTypes := map[string]bool{
		"application/pdf": true,
		"image/jpeg":      true,
		"image/jpg":       true,
		"image/png":       true,
	}
	return allowedTypes[mimeType]
}
