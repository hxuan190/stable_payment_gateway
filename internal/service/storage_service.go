package service

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

// StorageService handles file storage for KYC documents
type StorageService interface {
	UploadKYCDocument(ctx context.Context, merchantID uuid.UUID, docType model.KYCDocumentType, fileName string, data []byte) (string, error)
	GetDocumentURL(ctx context.Context, filePath string) (string, error)
	DeleteDocument(ctx context.Context, filePath string) error
}

type storageService struct {
	storageBackend StorageBackend
	basePath       string
}

// NewStorageService creates a new storage service
func NewStorageService(backend StorageBackend, basePath string) StorageService {
	return &storageService{
		storageBackend: backend,
		basePath:       basePath,
	}
}

func (s *storageService) UploadKYCDocument(ctx context.Context, merchantID uuid.UUID, docType model.KYCDocumentType, fileName string, data []byte) (string, error) {
	// Generate a structured file path: kyc/{merchant_id}/{doc_type}/{timestamp}_{filename}
	timestamp := time.Now().Format("20060102-150405")
	safeFileName := sanitizeFileName(fileName)
	filePath := filepath.Join(
		s.basePath,
		"kyc",
		merchantID.String(),
		string(docType),
		fmt.Sprintf("%s_%s", timestamp, safeFileName),
	)

	// Upload to storage backend (S3, MinIO, local filesystem, etc.)
	if err := s.storageBackend.Upload(ctx, filePath, data); err != nil {
		return "", fmt.Errorf("failed to upload to storage: %w", err)
	}

	return filePath, nil
}

func (s *storageService) GetDocumentURL(ctx context.Context, filePath string) (string, error) {
	// Generate a signed URL for accessing the document (e.g., S3 presigned URL)
	return s.storageBackend.GetSignedURL(ctx, filePath, 3600) // 1 hour expiry
}

func (s *storageService) DeleteDocument(ctx context.Context, filePath string) error {
	return s.storageBackend.Delete(ctx, filePath)
}

// ===== Storage Backend Interface =====

// StorageBackend is the interface for different storage implementations
type StorageBackend interface {
	Upload(ctx context.Context, path string, data []byte) error
	Download(ctx context.Context, path string) ([]byte, error)
	Delete(ctx context.Context, path string) error
	GetSignedURL(ctx context.Context, path string, expirySeconds int) (string, error)
}

// ===== Local Filesystem Implementation (for MVP/testing) =====

type LocalStorageBackend struct {
	rootDir string
}

func NewLocalStorageBackend(rootDir string) StorageBackend {
	return &LocalStorageBackend{rootDir: rootDir}
}

func (l *LocalStorageBackend) Upload(ctx context.Context, path string, data []byte) error {
	// TODO: Implement local file system upload
	// For MVP, this is a placeholder
	return nil
}

func (l *LocalStorageBackend) Download(ctx context.Context, path string) ([]byte, error) {
	// TODO: Implement local file system download
	return nil, nil
}

func (l *LocalStorageBackend) Delete(ctx context.Context, path string) error {
	// TODO: Implement local file system delete
	return nil
}

func (l *LocalStorageBackend) GetSignedURL(ctx context.Context, path string, expirySeconds int) (string, error) {
	// For local storage, return a direct path or localhost URL
	return fmt.Sprintf("http://localhost:8080/api/v1/kyc/documents/%s", path), nil
}

// ===== S3 Implementation (for production) =====

type S3StorageBackend struct {
	bucketName string
	region     string
	// Add AWS SDK client here
}

func NewS3StorageBackend(bucketName, region string) StorageBackend {
	return &S3StorageBackend{
		bucketName: bucketName,
		region:     region,
	}
}

func (s *S3StorageBackend) Upload(ctx context.Context, path string, data []byte) error {
	// TODO: Implement S3 upload using AWS SDK
	// Use aws-sdk-go-v2
	return nil
}

func (s *S3StorageBackend) Download(ctx context.Context, path string) ([]byte, error) {
	// TODO: Implement S3 download
	return nil, nil
}

func (s *S3StorageBackend) Delete(ctx context.Context, path string) error {
	// TODO: Implement S3 delete
	return nil
}

func (s *S3StorageBackend) GetSignedURL(ctx context.Context, path string, expirySeconds int) (string, error) {
	// TODO: Implement S3 presigned URL generation
	return "", nil
}

// ===== Helper Functions =====

func sanitizeFileName(fileName string) string {
	// Remove any potentially dangerous characters from filename
	// For now, just return the fileName - implement proper sanitization for production
	return fileName
}
