package interfaces

import (
	"context"
	"io"
)

// FileMetadata contains file metadata
type FileMetadata struct {
	Filename    string
	ContentType string
	Size        int64
	URL         string
	Key         string
}

// StorageService provides file storage capabilities
type StorageService interface {
	// Upload uploads a file to storage
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) (*FileMetadata, error)

	// Download downloads a file from storage
	Download(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete deletes a file from storage
	Delete(ctx context.Context, key string) error

	// GetURL gets a temporary URL for a file
	GetURL(ctx context.Context, key string, expiryMinutes int) (string, error)
}
