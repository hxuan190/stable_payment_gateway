package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/hxuan190/stable_payment_gateway/internal/api/dto"
	"github.com/hxuan190/stable_payment_gateway/internal/api/middleware"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
)

// StorageService defines the interface for file storage operations
type StorageService interface {
	UploadKYCDocument(ctx context.Context, merchantID string, docType string, filename string, content []byte) (string, error)
	DeleteFile(ctx context.Context, fileURL string) error
}

// KYCDocumentRepository defines the interface for KYC document data access
type KYCDocumentRepository interface {
	Create(ctx context.Context, doc *model.KYCDocument) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.KYCDocument, error)
	GetByMerchantID(ctx context.Context, merchantID string) ([]*model.KYCDocument, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// KYCHandler handles HTTP requests for KYC document operations
type KYCHandler struct {
	storageService StorageService
	kycDocRepo     KYCDocumentRepository
}

// NewKYCHandler creates a new KYC handler
func NewKYCHandler(storageService StorageService, kycDocRepo KYCDocumentRepository) *KYCHandler {
	return &KYCHandler{
		storageService: storageService,
		kycDocRepo:     kycDocRepo,
	}
}

// UploadDocument handles POST /api/v1/merchants/kyc/documents
// @Summary Upload a KYC document
// @Description Upload a KYC document for merchant verification
// @Tags kyc
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "KYC document file (PDF, JPG, PNG)"
// @Param document_type formData string true "Document type" Enums(business_registration, tax_certificate, bank_statement, id_card, utility_bill)
// @Success 201 {object} dto.APIResponse{data=dto.UploadKYCDocumentResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 401 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/merchants/kyc/documents [post]
// @Security ApiKeyAuth
func (h *KYCHandler) UploadDocument(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated merchant from context
	merchant, err := middleware.GetMerchantFromContext(c)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to get merchant from context")

		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("UNAUTHORIZED", "Authentication required"))
		return
	}

	// Parse multipart form
	var req dto.UploadKYCDocumentRequest
	if err := c.ShouldBind(&req); err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
		}).Warn("Invalid KYC document upload request")

		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
			"INVALID_REQUEST",
			"Invalid request parameters",
			err.Error(),
		))
		return
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
		}).Warn("No file uploaded")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"FILE_REQUIRED",
			"KYC document file is required",
		))
		return
	}

	// Validate file size (max 10MB)
	maxSizeBytes := int64(10 * 1024 * 1024) // 10MB
	if file.Size > maxSizeBytes {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"file_size":   file.Size,
			"max_size":    maxSizeBytes,
			"merchant_id": merchant.ID,
		}).Warn("File size exceeds limit")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"FILE_TOO_LARGE",
			fmt.Sprintf("File size exceeds maximum limit of %d MB", maxSizeBytes/(1024*1024)),
		))
		return
	}

	// Validate file type by extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExtensions := map[string]string{
		".pdf":  "application/pdf",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
	}

	mimeType, allowed := allowedExtensions[ext]
	if !allowed {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"file_extension": ext,
			"merchant_id":    merchant.ID,
		}).Warn("Invalid file type")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_FILE_TYPE",
			"Only PDF, JPG, and PNG files are allowed",
		))
		return
	}

	// Open and read file content
	fileContent, err := file.Open()
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
		}).Error("Failed to open uploaded file")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"FILE_READ_ERROR",
			"Failed to read uploaded file",
		))
		return
	}
	defer fileContent.Close()

	// Read file bytes
	fileBytes := make([]byte, file.Size)
	_, err = fileContent.Read(fileBytes)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
		}).Error("Failed to read file content")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"FILE_READ_ERROR",
			"Failed to read file content",
		))
		return
	}

	// Upload to storage (S3/MinIO)
	fileURL, err := h.storageService.UploadKYCDocument(ctx, merchant.ID, req.DocumentType, file.Filename, fileBytes)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":         err.Error(),
			"merchant_id":   merchant.ID,
			"document_type": req.DocumentType,
		}).Error("Failed to upload file to storage")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"STORAGE_ERROR",
			"Failed to upload document to storage",
		))
		return
	}

	// Create database record
	kycDoc := &model.KYCDocument{
		ID:         uuid.New().String(),
		MerchantID: merchant.ID,
		DocumentType: req.DocumentType,
		FileURL:    fileURL,
		FileSizeBytes: sql.NullInt64{
			Int64: file.Size,
			Valid: true,
		},
		MimeType: sql.NullString{
			String: mimeType,
			Valid:  true,
		},
		Status: model.KYCDocumentStatusPending,
	}

	err = h.kycDocRepo.Create(ctx, kycDoc)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
		}).Error("Failed to create KYC document record")

		// Try to clean up uploaded file
		_ = h.storageService.DeleteFile(ctx, fileURL)

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"DATABASE_ERROR",
			"Failed to save document record",
		))
		return
	}

	// Build response
	response := dto.KYCDocumentToResponse(kycDoc)

	logger.WithContext(ctx).WithFields(logrus.Fields{
		"document_id":   kycDoc.ID,
		"merchant_id":   merchant.ID,
		"document_type": req.DocumentType,
		"file_size":     file.Size,
	}).Info("KYC document uploaded successfully")

	c.JSON(http.StatusCreated, dto.SuccessResponse(response))
}

// ListDocuments handles GET /api/v1/merchants/kyc/documents
// @Summary List KYC documents
// @Description List all KYC documents for the authenticated merchant
// @Tags kyc
// @Accept json
// @Produce json
// @Success 200 {object} dto.APIResponse{data=dto.ListKYCDocumentsResponse}
// @Failure 401 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/merchants/kyc/documents [get]
// @Security ApiKeyAuth
func (h *KYCHandler) ListDocuments(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated merchant from context
	merchant, err := middleware.GetMerchantFromContext(c)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to get merchant from context")

		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("UNAUTHORIZED", "Authentication required"))
		return
	}

	// Get documents from repository
	documents, err := h.kycDocRepo.GetByMerchantID(ctx, merchant.ID)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
		}).Error("Failed to list KYC documents")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"DATABASE_ERROR",
			"Failed to retrieve documents",
		))
		return
	}

	// Convert to DTOs
	documentItems := make([]dto.KYCDocumentItem, len(documents))
	for i, doc := range documents {
		documentItems[i] = dto.KYCDocumentToListItem(doc)
	}

	response := dto.ListKYCDocumentsResponse{
		Documents: documentItems,
	}

	logger.WithContext(ctx).WithFields(logrus.Fields{
		"merchant_id": merchant.ID,
		"count":       len(documents),
	}).Debug("KYC documents listed successfully")

	c.JSON(http.StatusOK, dto.SuccessResponse(response))
}

// DeleteDocument handles DELETE /api/v1/merchants/kyc/documents/:id
// @Summary Delete a KYC document
// @Description Delete a KYC document (only if status is pending or rejected)
// @Tags kyc
// @Accept json
// @Produce json
// @Param id path string true "Document ID"
// @Success 200 {object} dto.APIResponse
// @Failure 400 {object} dto.APIResponse
// @Failure 401 {object} dto.APIResponse
// @Failure 403 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/merchants/kyc/documents/{id} [delete]
// @Security ApiKeyAuth
func (h *KYCHandler) DeleteDocument(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated merchant from context
	merchant, err := middleware.GetMerchantFromContext(c)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to get merchant from context")

		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("UNAUTHORIZED", "Authentication required"))
		return
	}

	// Get document ID from path parameter
	documentIDStr := c.Param("id")
	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("INVALID_REQUEST", "Invalid document ID"))
		return
	}

	// Get document from repository
	doc, err := h.kycDocRepo.GetByID(ctx, documentID)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"document_id": documentIDStr,
		}).Error("Failed to get KYC document")

		c.JSON(http.StatusNotFound, dto.ErrorResponse("DOCUMENT_NOT_FOUND", "Document not found"))
		return
	}

	// Verify document belongs to this merchant
	if doc.MerchantID != merchant.ID {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"merchant_id":         merchant.ID,
			"document_id":         documentIDStr,
			"document_merchant_id": doc.MerchantID,
		}).Warn("Merchant attempted to delete document from different merchant")

		c.JSON(http.StatusForbidden, dto.ErrorResponse("FORBIDDEN", "Access denied to this document"))
		return
	}

	// Only allow deletion if document is pending or rejected
	if doc.Status == model.KYCDocumentStatusApproved {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"document_id": documentIDStr,
			"status":      doc.Status,
		}).Warn("Cannot delete approved document")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"DOCUMENT_APPROVED",
			"Cannot delete an approved document",
		))
		return
	}

	// Delete from storage
	err = h.storageService.DeleteFile(ctx, doc.FileURL)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"document_id": documentIDStr,
			"file_url":    doc.FileURL,
		}).Warn("Failed to delete file from storage")
		// Continue anyway - we still want to delete the database record
	}

	// Delete from database
	err = h.kycDocRepo.Delete(ctx, documentID)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"document_id": documentIDStr,
		}).Error("Failed to delete KYC document record")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"DATABASE_ERROR",
			"Failed to delete document",
		))
		return
	}

	logger.WithContext(ctx).WithFields(logrus.Fields{
		"document_id": documentIDStr,
		"merchant_id": merchant.ID,
	}).Info("KYC document deleted successfully")

	c.JSON(http.StatusOK, dto.SuccessResponse(map[string]string{
		"message": "Document deleted successfully",
	}))
}
