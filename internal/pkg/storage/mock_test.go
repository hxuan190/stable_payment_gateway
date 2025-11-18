package storage

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockStorage_UploadAndDownloadKYCDocument(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	// Prepare test data
	merchantID := "merchant-123"
	docType := "business_registration"
	filename := "license.pdf"
	fileData := []byte("test file content")
	contentType := "application/pdf"

	// Upload file
	fileURL, err := storage.UploadKYCDocument(ctx, merchantID, docType, filename, bytes.NewReader(fileData), contentType)
	require.NoError(t, err)
	assert.NotEmpty(t, fileURL)
	assert.Contains(t, fileURL, "mock://storage/")

	// Verify file count
	assert.Equal(t, 1, storage.GetFileCount())

	// Download file
	downloadedFile, err := storage.DownloadKYCDocument(ctx, fileURL)
	require.NoError(t, err)
	defer downloadedFile.Close()

	// Read downloaded data
	downloadedData, err := io.ReadAll(downloadedFile)
	require.NoError(t, err)

	// Verify content
	assert.Equal(t, fileData, downloadedData)
}

func TestMockStorage_DeleteKYCDocument(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	// Upload a file
	fileURL, err := storage.UploadKYCDocument(ctx, "merchant-123", "business_registration", "license.pdf", bytes.NewReader([]byte("test")), "application/pdf")
	require.NoError(t, err)

	// Verify file exists
	assert.Equal(t, 1, storage.GetFileCount())

	// Delete file
	err = storage.DeleteKYCDocument(ctx, fileURL)
	require.NoError(t, err)

	// Verify file is deleted
	assert.Equal(t, 0, storage.GetFileCount())

	// Try to download deleted file (should fail)
	_, err = storage.DownloadKYCDocument(ctx, fileURL)
	assert.Error(t, err)
}

func TestMockStorage_ArchiveAndGetAuditLogs(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	// Archive audit logs
	year := 2025
	month := 11
	data := []byte("compressed audit log data")

	fileURL, err := storage.ArchiveAuditLogs(ctx, year, month, data)
	require.NoError(t, err)
	assert.NotEmpty(t, fileURL)
	assert.Contains(t, fileURL, "audit-logs")

	// Retrieve audit logs
	downloadedFile, err := storage.GetArchivedAuditLogs(ctx, year, month)
	require.NoError(t, err)
	defer downloadedFile.Close()

	// Read data
	downloadedData, err := io.ReadAll(downloadedFile)
	require.NoError(t, err)

	// Verify content
	assert.Equal(t, data, downloadedData)
}

func TestMockStorage_UploadAndDownloadFile(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	bucket := "test-bucket"
	key := "test-file.txt"
	fileData := []byte("test file content")
	contentType := "text/plain"

	// Upload file
	fileURL, err := storage.UploadFile(ctx, bucket, key, bytes.NewReader(fileData), contentType)
	require.NoError(t, err)
	assert.NotEmpty(t, fileURL)

	// Download file
	downloadedFile, err := storage.DownloadFile(ctx, bucket, key)
	require.NoError(t, err)
	defer downloadedFile.Close()

	// Read data
	downloadedData, err := io.ReadAll(downloadedFile)
	require.NoError(t, err)

	// Verify content
	assert.Equal(t, fileData, downloadedData)
}

func TestMockStorage_FileExists(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	bucket := "test-bucket"
	key := "test-file.txt"

	// Check non-existent file
	exists, err := storage.FileExists(ctx, bucket, key)
	require.NoError(t, err)
	assert.False(t, exists)

	// Upload file
	_, err = storage.UploadFile(ctx, bucket, key, bytes.NewReader([]byte("test")), "text/plain")
	require.NoError(t, err)

	// Check existing file
	exists, err = storage.FileExists(ctx, bucket, key)
	require.NoError(t, err)
	assert.True(t, exists)

	// Delete file
	err = storage.DeleteFile(ctx, bucket, key)
	require.NoError(t, err)

	// Check deleted file
	exists, err = storage.FileExists(ctx, bucket, key)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestMockStorage_GetFileMetadata(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	bucket := "test-bucket"
	key := "test-file.txt"
	fileData := []byte("test file content")
	contentType := "text/plain"

	// Upload file
	_, err := storage.UploadFile(ctx, bucket, key, bytes.NewReader(fileData), contentType)
	require.NoError(t, err)

	// Get metadata
	metadata, err := storage.GetFileMetadata(ctx, bucket, key)
	require.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, key, metadata.Key)
	assert.Equal(t, int64(len(fileData)), metadata.Size)
	assert.Equal(t, contentType, metadata.ContentType)
	assert.NotEmpty(t, metadata.ETag)
	assert.False(t, metadata.LastModified.IsZero())
}

func TestMockStorage_Clear(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	// Upload multiple files
	_, err := storage.UploadFile(ctx, "bucket1", "file1.txt", bytes.NewReader([]byte("test1")), "text/plain")
	require.NoError(t, err)
	_, err = storage.UploadFile(ctx, "bucket1", "file2.txt", bytes.NewReader([]byte("test2")), "text/plain")
	require.NoError(t, err)
	_, err = storage.UploadFile(ctx, "bucket2", "file3.txt", bytes.NewReader([]byte("test3")), "text/plain")
	require.NoError(t, err)

	// Verify files exist
	assert.Equal(t, 3, storage.GetFileCount())

	// Clear storage
	storage.Clear()

	// Verify all files are gone
	assert.Equal(t, 0, storage.GetFileCount())
}

func TestMockStorage_ListFiles(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	// Upload multiple files
	_, err := storage.UploadFile(ctx, "bucket1", "file1.txt", bytes.NewReader([]byte("test1")), "text/plain")
	require.NoError(t, err)
	_, err = storage.UploadFile(ctx, "bucket1", "file2.txt", bytes.NewReader([]byte("test2")), "text/plain")
	require.NoError(t, err)

	// List files
	files := storage.ListFiles()
	assert.Len(t, files, 2)
	assert.Contains(t, files, "bucket1/file1.txt")
	assert.Contains(t, files, "bucket1/file2.txt")
}

func TestMockStorage_DownloadNonExistentFile(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	// Try to download non-existent file
	_, err := storage.DownloadFile(ctx, "bucket", "nonexistent.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestMockStorage_GetMetadataForNonExistentFile(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	// Try to get metadata for non-existent file
	_, err := storage.GetFileMetadata(ctx, "bucket", "nonexistent.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}
