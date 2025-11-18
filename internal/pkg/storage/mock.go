package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MockStorage is an in-memory implementation of StorageService for testing
type MockStorage struct {
	files map[string]*MockFile
	mu    sync.RWMutex
}

// MockFile represents a file stored in memory
type MockFile struct {
	Key          string
	Data         []byte
	ContentType  string
	UploadedAt   time.Time
	Metadata     map[string]string
}

// NewMockStorage creates a new mock storage service
func NewMockStorage() *MockStorage {
	return &MockStorage{
		files: make(map[string]*MockFile),
	}
}

// UploadKYCDocument uploads a KYC document to mock storage
func (m *MockStorage) UploadKYCDocument(ctx context.Context, merchantID string, docType string, filename string, file io.Reader, contentType string) (string, error) {
	fileID := uuid.New().String()
	ext := filepath.Ext(filename)
	key := fmt.Sprintf("kyc/%s/%s/%s%s", merchantID, docType, fileID, ext)

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Store in memory
	m.mu.Lock()
	m.files[key] = &MockFile{
		Key:         key,
		Data:        data,
		ContentType: contentType,
		UploadedAt:  time.Now(),
		Metadata:    make(map[string]string),
	}
	m.mu.Unlock()

	// Return mock URL
	fileURL := fmt.Sprintf("mock://storage/%s", key)
	return fileURL, nil
}

// DownloadKYCDocument downloads a KYC document from mock storage
func (m *MockStorage) DownloadKYCDocument(ctx context.Context, fileURL string) (io.ReadCloser, error) {
	// Extract key from URL
	key := fileURL[len("mock://storage/"):]

	m.mu.RLock()
	file, exists := m.files[key]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("file not found: %s", key)
	}

	// Return data as ReadCloser
	return io.NopCloser(bytes.NewReader(file.Data)), nil
}

// DeleteKYCDocument deletes a KYC document from mock storage
func (m *MockStorage) DeleteKYCDocument(ctx context.Context, fileURL string) error {
	// Extract key from URL
	key := fileURL[len("mock://storage/"):]

	m.mu.Lock()
	delete(m.files, key)
	m.mu.Unlock()

	return nil
}

// ArchiveAuditLogs archives audit logs to mock storage
func (m *MockStorage) ArchiveAuditLogs(ctx context.Context, year int, month int, data []byte) (string, error) {
	key := fmt.Sprintf("audit-logs/%d/%02d/audit-logs.json.gz", year, month)

	m.mu.Lock()
	m.files[key] = &MockFile{
		Key:         key,
		Data:        data,
		ContentType: "application/gzip",
		UploadedAt:  time.Now(),
		Metadata:    make(map[string]string),
	}
	m.mu.Unlock()

	fileURL := fmt.Sprintf("mock://storage/%s", key)
	return fileURL, nil
}

// GetArchivedAuditLogs retrieves archived audit logs from mock storage
func (m *MockStorage) GetArchivedAuditLogs(ctx context.Context, year int, month int) (io.ReadCloser, error) {
	key := fmt.Sprintf("audit-logs/%d/%02d/audit-logs.json.gz", year, month)

	m.mu.RLock()
	file, exists := m.files[key]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("audit logs not found for %d-%02d", year, month)
	}

	return io.NopCloser(bytes.NewReader(file.Data)), nil
}

// UploadFile uploads a file to mock storage
func (m *MockStorage) UploadFile(ctx context.Context, bucket string, key string, file io.Reader, contentType string) (string, error) {
	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	fullKey := fmt.Sprintf("%s/%s", bucket, key)

	// Store in memory
	m.mu.Lock()
	m.files[fullKey] = &MockFile{
		Key:         fullKey,
		Data:        data,
		ContentType: contentType,
		UploadedAt:  time.Now(),
		Metadata:    make(map[string]string),
	}
	m.mu.Unlock()

	// Return mock URL
	fileURL := fmt.Sprintf("mock://storage/%s", fullKey)
	return fileURL, nil
}

// DownloadFile downloads a file from mock storage
func (m *MockStorage) DownloadFile(ctx context.Context, bucket string, key string) (io.ReadCloser, error) {
	fullKey := fmt.Sprintf("%s/%s", bucket, key)

	m.mu.RLock()
	file, exists := m.files[fullKey]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("file not found: %s", fullKey)
	}

	return io.NopCloser(bytes.NewReader(file.Data)), nil
}

// DeleteFile deletes a file from mock storage
func (m *MockStorage) DeleteFile(ctx context.Context, bucket string, key string) error {
	fullKey := fmt.Sprintf("%s/%s", bucket, key)

	m.mu.Lock()
	delete(m.files, fullKey)
	m.mu.Unlock()

	return nil
}

// FileExists checks if a file exists in mock storage
func (m *MockStorage) FileExists(ctx context.Context, bucket string, key string) (bool, error) {
	fullKey := fmt.Sprintf("%s/%s", bucket, key)

	m.mu.RLock()
	_, exists := m.files[fullKey]
	m.mu.RUnlock()

	return exists, nil
}

// GetFileMetadata retrieves metadata about a file in mock storage
func (m *MockStorage) GetFileMetadata(ctx context.Context, bucket string, key string) (*FileMetadata, error) {
	fullKey := fmt.Sprintf("%s/%s", bucket, key)

	m.mu.RLock()
	file, exists := m.files[fullKey]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("file not found: %s", fullKey)
	}

	metadata := &FileMetadata{
		Key:          key,
		Size:         int64(len(file.Data)),
		ContentType:  file.ContentType,
		ETag:         fmt.Sprintf("\"%x\"", uuid.New()),
		LastModified: file.UploadedAt,
		Metadata:     file.Metadata,
	}

	return metadata, nil
}

// Clear removes all files from mock storage (useful for tests)
func (m *MockStorage) Clear() {
	m.mu.Lock()
	m.files = make(map[string]*MockFile)
	m.mu.Unlock()
}

// GetFileCount returns the number of files in mock storage (useful for tests)
func (m *MockStorage) GetFileCount() int {
	m.mu.RLock()
	count := len(m.files)
	m.mu.RUnlock()
	return count
}

// ListFiles returns all file keys in mock storage (useful for tests)
func (m *MockStorage) ListFiles() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.files))
	for key := range m.files {
		keys = append(keys, key)
	}
	return keys
}
