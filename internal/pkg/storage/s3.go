package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

// S3Storage implements StorageService using AWS S3
type S3Storage struct {
	client         *s3.Client
	uploader       *manager.Uploader
	downloader     *manager.Downloader
	kycBucket      string
	auditBucket    string
	region         string
	encryption     string // "AES256" or "aws:kms"
}

// S3Config represents configuration for S3 storage
type S3Config struct {
	Region         string
	KYCBucket      string
	AuditBucket    string
	Encryption     string // "AES256" or "aws:kms"
	KMSKeyID       string // Optional: KMS key ID for encryption
}

// NewS3Storage creates a new S3 storage service
func NewS3Storage(ctx context.Context, cfg S3Config) (*S3Storage, error) {
	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg)

	// Create uploader and downloader
	uploader := manager.NewUploader(client)
	downloader := manager.NewDownloader(client)

	encryption := cfg.Encryption
	if encryption == "" {
		encryption = "AES256" // Default to AES256
	}

	return &S3Storage{
		client:      client,
		uploader:    uploader,
		downloader:  downloader,
		kycBucket:   cfg.KYCBucket,
		auditBucket: cfg.AuditBucket,
		region:      cfg.Region,
		encryption:  encryption,
	}, nil
}

// UploadKYCDocument uploads a KYC document to S3
func (s *S3Storage) UploadKYCDocument(ctx context.Context, merchantID string, docType string, filename string, file io.Reader, contentType string) (string, error) {
	// Generate unique key
	fileID := uuid.New().String()
	ext := filepath.Ext(filename)
	key := fmt.Sprintf("kyc/%s/%s/%s%s", merchantID, docType, fileID, ext)

	// Upload to KYC bucket
	fileURL, err := s.UploadFile(ctx, s.kycBucket, key, file, contentType)
	if err != nil {
		return "", fmt.Errorf("failed to upload KYC document: %w", err)
	}

	return fileURL, nil
}

// DownloadKYCDocument downloads a KYC document from S3
func (s *S3Storage) DownloadKYCDocument(ctx context.Context, fileURL string) (io.ReadCloser, error) {
	// Extract key from URL
	key, err := s.extractKeyFromURL(fileURL)
	if err != nil {
		return nil, err
	}

	return s.DownloadFile(ctx, s.kycBucket, key)
}

// DeleteKYCDocument deletes a KYC document from S3
func (s *S3Storage) DeleteKYCDocument(ctx context.Context, fileURL string) error {
	// Extract key from URL
	key, err := s.extractKeyFromURL(fileURL)
	if err != nil {
		return err
	}

	return s.DeleteFile(ctx, s.kycBucket, key)
}

// ArchiveAuditLogs archives audit logs to S3 (with Glacier lifecycle)
func (s *S3Storage) ArchiveAuditLogs(ctx context.Context, year int, month int, data []byte) (string, error) {
	// Generate key for audit logs
	key := fmt.Sprintf("audit-logs/%d/%02d/audit-logs.json.gz", year, month)

	// Upload to audit bucket
	reader := bytes.NewReader(data)
	fileURL, err := s.UploadFile(ctx, s.auditBucket, key, reader, "application/gzip")
	if err != nil {
		return "", fmt.Errorf("failed to archive audit logs: %w", err)
	}

	return fileURL, nil
}

// GetArchivedAuditLogs retrieves archived audit logs from S3
func (s *S3Storage) GetArchivedAuditLogs(ctx context.Context, year int, month int) (io.ReadCloser, error) {
	key := fmt.Sprintf("audit-logs/%d/%02d/audit-logs.json.gz", year, month)
	return s.DownloadFile(ctx, s.auditBucket, key)
}

// UploadFile uploads a file to S3
func (s *S3Storage) UploadFile(ctx context.Context, bucket string, key string, file io.Reader, contentType string) (string, error) {
	// Prepare upload input
	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	}

	// Add server-side encryption
	if s.encryption == "AES256" {
		input.ServerSideEncryption = types.ServerSideEncryptionAes256
	}

	// Upload the file
	_, err := s.uploader.Upload(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// Generate file URL
	fileURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, s.region, key)

	return fileURL, nil
}

// DownloadFile downloads a file from S3
func (s *S3Storage) DownloadFile(ctx context.Context, bucket string, key string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download file from S3: %w", err)
	}

	return result.Body, nil
}

// DeleteFile deletes a file from S3
func (s *S3Storage) DeleteFile(ctx context.Context, bucket string, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

// FileExists checks if a file exists in S3
func (s *S3Storage) FileExists(ctx context.Context, bucket string, key string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.HeadObject(ctx, input)
	if err != nil {
		// Check if error is NoSuchKey
		// If so, file doesn't exist
		return false, nil
	}

	return true, nil
}

// GetFileMetadata retrieves metadata about a file in S3
func (s *S3Storage) GetFileMetadata(ctx context.Context, bucket string, key string) (*FileMetadata, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}

	metadata := &FileMetadata{
		Key:          key,
		Size:         *result.ContentLength,
		ContentType:  *result.ContentType,
		ETag:         *result.ETag,
		LastModified: *result.LastModified,
		Metadata:     result.Metadata,
	}

	return metadata, nil
}

// extractKeyFromURL extracts the S3 key from a full S3 URL
func (s *S3Storage) extractKeyFromURL(url string) (string, error) {
	// Simple implementation - in production, use proper URL parsing
	// Expected format: https://bucket.s3.region.amazonaws.com/key
	// For now, just return the URL as-is if it doesn't start with https
	if len(url) < 8 || url[:8] != "https://" {
		return url, nil
	}

	// TODO: Implement proper URL parsing
	return "", fmt.Errorf("URL parsing not implemented yet")
}

// SetLifecyclePolicy sets a lifecycle policy on the bucket to transition to Glacier
// This should be called during initialization to ensure audit logs are archived to Glacier
func (s *S3Storage) SetLifecyclePolicy(ctx context.Context, bucket string, daysToGlacier int, daysToExpiration int) error {
	input := &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucket),
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{
			Rules: []types.LifecycleRule{
				{
					ID:     aws.String("archive-to-glacier"),  // Fixed: ID instead of Id
					Prefix: aws.String("audit-logs/"),          // Fixed: Use Prefix directly (deprecated but simpler)
					Status: types.ExpirationStatusEnabled,
					Transitions: []types.Transition{
						{
							Days:         aws.Int32(int32(daysToGlacier)),
							StorageClass: types.TransitionStorageClassGlacier,
						},
					},
				},
			},
		},
	}

	if daysToExpiration > 0 {
		input.LifecycleConfiguration.Rules[0].Expiration = &types.LifecycleExpiration{
			Days: aws.Int32(int32(daysToExpiration)),
		}
	}

	_, err := s.client.PutBucketLifecycleConfiguration(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to set lifecycle policy: %w", err)
	}

	return nil
}
