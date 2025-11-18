package storage

import (
	"context"
	"io"
	"time"
)

// StorageService defines the interface for file storage operations
type StorageService interface {
	// KYC Document operations
	UploadKYCDocument(ctx context.Context, merchantID string, docType string, filename string, file io.Reader, contentType string) (string, error)
	DownloadKYCDocument(ctx context.Context, fileURL string) (io.ReadCloser, error)
	DeleteKYCDocument(ctx context.Context, fileURL string) error

	// Audit log archival operations
	ArchiveAuditLogs(ctx context.Context, year int, month int, data []byte) (string, error)
	GetArchivedAuditLogs(ctx context.Context, year int, month int) (io.ReadCloser, error)

	// Generic file operations
	UploadFile(ctx context.Context, bucket string, key string, file io.Reader, contentType string) (string, error)
	DownloadFile(ctx context.Context, bucket string, key string) (io.ReadCloser, error)
	DeleteFile(ctx context.Context, bucket string, key string) error
	FileExists(ctx context.Context, bucket string, key string) (bool, error)
	GetFileMetadata(ctx context.Context, bucket string, key string) (*FileMetadata, error)
}

// FileMetadata represents metadata about a stored file
type FileMetadata struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
	Metadata     map[string]string
}
