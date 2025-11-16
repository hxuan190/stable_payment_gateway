package model

import (
	"time"

	"github.com/google/uuid"
)

// KYCDocumentType represents the type of KYC document
type KYCDocumentType string

const (
	DocumentTypeCCCDFront           KYCDocumentType = "cccd_front"            // CCCD mặt trước
	DocumentTypeCCCDBack            KYCDocumentType = "cccd_back"             // CCCD mặt sau
	DocumentTypeSelfie              KYCDocumentType = "selfie"                // Ảnh chân dung
	DocumentTypeBusinessLicense     KYCDocumentType = "business_license"      // Giấy phép kinh doanh (HKT)
	DocumentTypeBusinessRegistration KYCDocumentType = "business_registration" // Giấy đăng ký kinh doanh (Company)
	DocumentTypeBusinessCharter     KYCDocumentType = "business_charter"      // Điều lệ (Company)
	DocumentTypeDirectorID          KYCDocumentType = "director_id"           // CCCD giám đốc
	DocumentTypeAppointmentDecision KYCDocumentType = "appointment_decision"  // Quyết định bổ nhiệm
	DocumentTypeShopPhoto           KYCDocumentType = "shop_photo"            // Ảnh bảng hiệu
	DocumentTypeOther               KYCDocumentType = "other"
)

// KYCDocument represents a document uploaded for KYC verification
type KYCDocument struct {
	ID              uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	KYCSubmissionID uuid.UUID       `json:"kyc_submission_id" gorm:"type:uuid;not null"`
	MerchantID      uuid.UUID       `json:"merchant_id" gorm:"type:uuid;not null"`
	DocumentType    KYCDocumentType `json:"document_type" gorm:"type:varchar(50);not null"`

	// File information
	FilePath      string  `json:"file_path" gorm:"type:varchar(500);not null"`        // S3/MinIO path
	FileName      string  `json:"file_name" gorm:"type:varchar(255);not null"`
	FileSizeBytes *int64  `json:"file_size_bytes,omitempty" gorm:"type:bigint"`
	MimeType      *string `json:"mime_type,omitempty" gorm:"type:varchar(100)"`

	// Metadata extracted from document (OCR results, etc.)
	ExtractedData *string `json:"extracted_data,omitempty" gorm:"type:jsonb"`

	// Verification status
	Verified           bool       `json:"verified" gorm:"default:false"`
	VerifiedAt         *time.Time `json:"verified_at,omitempty"`
	VerificationMethod *string    `json:"verification_method,omitempty" gorm:"type:varchar(50)"` // 'ocr', 'manual', 'ai'

	UploadedAt time.Time `json:"uploaded_at" gorm:"default:now()"`
	CreatedAt  time.Time `json:"created_at" gorm:"default:now()"`

	// Relationships
	KYCSubmission *KYCSubmission `json:"kyc_submission,omitempty" gorm:"foreignKey:KYCSubmissionID"`
	Merchant      *Merchant      `json:"merchant,omitempty" gorm:"foreignKey:MerchantID"`
}

// TableName specifies the table name for GORM
func (KYCDocument) TableName() string {
	return "kyc_documents"
}

// IsVerified checks if the document has been verified
func (d *KYCDocument) IsVerified() bool {
	return d.Verified && d.VerifiedAt != nil
}

// GetExtractedDataMap returns the extracted data as a map
func (d *KYCDocument) GetExtractedDataMap() map[string]interface{} {
	// This would typically parse the JSONB field
	// For now, returning nil - implement JSON parsing as needed
	return nil
}
