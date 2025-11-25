package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

var (
	// ErrArchivedRecordNotFound is returned when an archived record is not found
	ErrArchivedRecordNotFound = errors.New("archived record not found")
)

// ArchivedRecordRepository handles database operations for archived records
type ArchivedRecordRepository struct {
	db *sql.DB
}

// NewArchivedRecordRepository creates a new archived record repository
func NewArchivedRecordRepository(db *sql.DB) *ArchivedRecordRepository {
	return &ArchivedRecordRepository{
		db: db,
	}
}

// Create inserts a new archived record
func (r *ArchivedRecordRepository) Create(record *model.ArchivedRecord) error {
	if record == nil {
		return errors.New("archived record cannot be nil")
	}

	// Generate UUID if not provided
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}

	query := `
		INSERT INTO archived_records (
			id, original_id, table_name,
			archive_path, s3_bucket, s3_region, storage_class,
			data_hash, compression_algorithm, encrypted, encryption_algorithm,
			original_size_bytes, compressed_size_bytes, object_size_bytes,
			archived_at, archived_by, archive_job_id,
			restore_initiated_at, restore_expires_at, restore_tier,
			restored_at, restore_initiated_by,
			record_data_sample, metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26
		)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = now
	}

	err := r.db.QueryRow(
		query,
		record.ID,
		record.OriginalID,
		record.OriginalTableName,
		record.ArchivePath,
		record.S3Bucket,
		record.S3Region,
		record.StorageClass,
		record.DataHash,
		record.CompressionAlgorithm,
		record.Encrypted,
		record.EncryptionAlgorithm,
		record.OriginalSizeBytes,
		record.CompressedSizeBytes,
		record.ObjectSizeBytes,
		record.ArchivedAt,
		record.ArchivedBy,
		record.ArchiveJobID,
		record.RestoreInitiatedAt,
		record.RestoreExpiresAt,
		record.RestoreTier,
		record.RestoredAt,
		record.RestoreInitiatedBy,
		record.RecordDataSample,
		record.Metadata,
		record.CreatedAt,
		record.UpdatedAt,
	).Scan(&record.ID, &record.CreatedAt, &record.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create archived record: %w", err)
	}

	return nil
}

// GetByID retrieves an archived record by ID
func (r *ArchivedRecordRepository) GetByID(id uuid.UUID) (*model.ArchivedRecord, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid archived record ID")
	}

	query := `
		SELECT
			id, original_id, table_name,
			archive_path, s3_bucket, s3_region, storage_class,
			data_hash, compression_algorithm, encrypted, encryption_algorithm,
			original_size_bytes, compressed_size_bytes, object_size_bytes,
			archived_at, archived_by, archive_job_id,
			restore_initiated_at, restore_expires_at, restore_tier,
			restored_at, restore_initiated_by,
			record_data_sample, metadata,
			created_at, updated_at
		FROM archived_records
		WHERE id = $1
	`

	record := &model.ArchivedRecord{}
	err := r.db.QueryRow(query, id).Scan(
		&record.ID,
		&record.OriginalID,
		&record.OriginalTableName,
		&record.ArchivePath,
		&record.S3Bucket,
		&record.S3Region,
		&record.StorageClass,
		&record.DataHash,
		&record.CompressionAlgorithm,
		&record.Encrypted,
		&record.EncryptionAlgorithm,
		&record.OriginalSizeBytes,
		&record.CompressedSizeBytes,
		&record.ObjectSizeBytes,
		&record.ArchivedAt,
		&record.ArchivedBy,
		&record.ArchiveJobID,
		&record.RestoreInitiatedAt,
		&record.RestoreExpiresAt,
		&record.RestoreTier,
		&record.RestoredAt,
		&record.RestoreInitiatedBy,
		&record.RecordDataSample,
		&record.Metadata,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrArchivedRecordNotFound
		}
		return nil, fmt.Errorf("failed to get archived record by ID: %w", err)
	}

	return record, nil
}

// GetByOriginalID retrieves an archived record by original ID and table name
func (r *ArchivedRecordRepository) GetByOriginalID(tableName string, originalID uuid.UUID) (*model.ArchivedRecord, error) {
	if tableName == "" {
		return nil, errors.New("table name cannot be empty")
	}
	if originalID == uuid.Nil {
		return nil, errors.New("original ID cannot be empty")
	}

	query := `
		SELECT
			id, original_id, table_name,
			archive_path, s3_bucket, s3_region, storage_class,
			data_hash, compression_algorithm, encrypted, encryption_algorithm,
			original_size_bytes, compressed_size_bytes, object_size_bytes,
			archived_at, archived_by, archive_job_id,
			restore_initiated_at, restore_expires_at, restore_tier,
			restored_at, restore_initiated_by,
			record_data_sample, metadata,
			created_at, updated_at
		FROM archived_records
		WHERE table_name = $1 AND original_id = $2
	`

	record := &model.ArchivedRecord{}
	err := r.db.QueryRow(query, tableName, originalID).Scan(
		&record.ID,
		&record.OriginalID,
		&record.OriginalTableName,
		&record.ArchivePath,
		&record.S3Bucket,
		&record.S3Region,
		&record.StorageClass,
		&record.DataHash,
		&record.CompressionAlgorithm,
		&record.Encrypted,
		&record.EncryptionAlgorithm,
		&record.OriginalSizeBytes,
		&record.CompressedSizeBytes,
		&record.ObjectSizeBytes,
		&record.ArchivedAt,
		&record.ArchivedBy,
		&record.ArchiveJobID,
		&record.RestoreInitiatedAt,
		&record.RestoreExpiresAt,
		&record.RestoreTier,
		&record.RestoredAt,
		&record.RestoreInitiatedBy,
		&record.RecordDataSample,
		&record.Metadata,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrArchivedRecordNotFound
		}
		return nil, fmt.Errorf("failed to get archived record by original ID: %w", err)
	}

	return record, nil
}

// ListByTableName retrieves archived records by table name with pagination
func (r *ArchivedRecordRepository) ListByTableName(tableName string, limit, offset int) ([]*model.ArchivedRecord, error) {
	if tableName == "" {
		return nil, errors.New("table name cannot be empty")
	}

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, original_id, table_name,
			archive_path, s3_bucket, s3_region, storage_class,
			data_hash, compression_algorithm, encrypted, encryption_algorithm,
			original_size_bytes, compressed_size_bytes, object_size_bytes,
			archived_at, archived_by, archive_job_id,
			restore_initiated_at, restore_expires_at, restore_tier,
			restored_at, restore_initiated_by,
			record_data_sample, metadata,
			created_at, updated_at
		FROM archived_records
		WHERE table_name = $1
		ORDER BY archived_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, tableName, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list archived records by table name: %w", err)
	}
	defer rows.Close()

	records := make([]*model.ArchivedRecord, 0)
	for rows.Next() {
		record := &model.ArchivedRecord{}
		err := rows.Scan(
			&record.ID,
			&record.OriginalID,
			&record.OriginalTableName,
			&record.ArchivePath,
			&record.S3Bucket,
			&record.S3Region,
			&record.StorageClass,
			&record.DataHash,
			&record.CompressionAlgorithm,
			&record.Encrypted,
			&record.EncryptionAlgorithm,
			&record.OriginalSizeBytes,
			&record.CompressedSizeBytes,
			&record.ObjectSizeBytes,
			&record.ArchivedAt,
			&record.ArchivedBy,
			&record.ArchiveJobID,
			&record.RestoreInitiatedAt,
			&record.RestoreExpiresAt,
			&record.RestoreTier,
			&record.RestoredAt,
			&record.RestoreInitiatedBy,
			&record.RecordDataSample,
			&record.Metadata,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan archived record: %w", err)
		}
		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating archived records: %w", err)
	}

	return records, nil
}

// Update updates an existing archived record
func (r *ArchivedRecordRepository) Update(record *model.ArchivedRecord) error {
	if record == nil {
		return errors.New("archived record cannot be nil")
	}
	if record.ID == uuid.Nil {
		return errors.New("invalid archived record ID")
	}

	query := `
		UPDATE archived_records SET
			restore_initiated_at = $2,
			restore_expires_at = $3,
			restore_tier = $4,
			restored_at = $5,
			restore_initiated_by = $6,
			metadata = $7,
			updated_at = $8
		WHERE id = $1
		RETURNING updated_at
	`

	record.UpdatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		record.ID,
		record.RestoreInitiatedAt,
		record.RestoreExpiresAt,
		record.RestoreTier,
		record.RestoredAt,
		record.RestoreInitiatedBy,
		record.Metadata,
		record.UpdatedAt,
	).Scan(&record.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrArchivedRecordNotFound
		}
		return fmt.Errorf("failed to update archived record: %w", err)
	}

	return nil
}

// InitiateRestore initiates a restore operation for an archived record
func (r *ArchivedRecordRepository) InitiateRestore(id uuid.UUID, tier model.RestoreTier, initiatedBy string) error {
	if id == uuid.Nil {
		return errors.New("invalid archived record ID")
	}

	query := `
		UPDATE archived_records
		SET
			restore_initiated_at = $2,
			restore_tier = $3,
			restore_initiated_by = $4,
			updated_at = $2
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(
		query,
		id,
		sql.NullTime{Time: now, Valid: true},
		sql.NullString{String: string(tier), Valid: true},
		sql.NullString{String: initiatedBy, Valid: true},
	)
	if err != nil {
		return fmt.Errorf("failed to initiate restore: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrArchivedRecordNotFound
	}

	return nil
}

// MarkAsRestored marks an archived record as restored
func (r *ArchivedRecordRepository) MarkAsRestored(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid archived record ID")
	}

	query := `
		UPDATE archived_records
		SET
			restored_at = $2,
			updated_at = $2
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(query, id, sql.NullTime{Time: now, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to mark as restored: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrArchivedRecordNotFound
	}

	return nil
}

// ListPendingRestores retrieves archived records with pending restore operations
func (r *ArchivedRecordRepository) ListPendingRestores() ([]*model.ArchivedRecord, error) {
	query := `
		SELECT
			id, original_id, table_name,
			archive_path, s3_bucket, s3_region, storage_class,
			data_hash, compression_algorithm, encrypted, encryption_algorithm,
			original_size_bytes, compressed_size_bytes, object_size_bytes,
			archived_at, archived_by, archive_job_id,
			restore_initiated_at, restore_expires_at, restore_tier,
			restored_at, restore_initiated_by,
			record_data_sample, metadata,
			created_at, updated_at
		FROM archived_records
		WHERE restore_initiated_at IS NOT NULL
		  AND (restored_at IS NULL OR restored_at > CURRENT_TIMESTAMP)
		  AND (restore_expires_at IS NULL OR restore_expires_at > CURRENT_TIMESTAMP)
		ORDER BY restore_initiated_at ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending restores: %w", err)
	}
	defer rows.Close()

	records := make([]*model.ArchivedRecord, 0)
	for rows.Next() {
		record := &model.ArchivedRecord{}
		err := rows.Scan(
			&record.ID,
			&record.OriginalID,
			&record.OriginalTableName,
			&record.ArchivePath,
			&record.S3Bucket,
			&record.S3Region,
			&record.StorageClass,
			&record.DataHash,
			&record.CompressionAlgorithm,
			&record.Encrypted,
			&record.EncryptionAlgorithm,
			&record.OriginalSizeBytes,
			&record.CompressedSizeBytes,
			&record.ObjectSizeBytes,
			&record.ArchivedAt,
			&record.ArchivedBy,
			&record.ArchiveJobID,
			&record.RestoreInitiatedAt,
			&record.RestoreExpiresAt,
			&record.RestoreTier,
			&record.RestoredAt,
			&record.RestoreInitiatedBy,
			&record.RecordDataSample,
			&record.Metadata,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan archived record: %w", err)
		}
		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating archived records: %w", err)
	}

	return records, nil
}

// GetStorageStats retrieves storage statistics by table name
func (r *ArchivedRecordRepository) GetStorageStats(tableName string) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_records,
			SUM(original_size_bytes) as total_original_bytes,
			SUM(compressed_size_bytes) as total_compressed_bytes,
			SUM(object_size_bytes) as total_storage_bytes,
			MIN(archived_at) as first_archived_at,
			MAX(archived_at) as last_archived_at
		FROM archived_records
		WHERE table_name = $1
	`

	var stats map[string]interface{} = make(map[string]interface{})
	var totalRecords int64
	var totalOriginalBytes, totalCompressedBytes, totalStorageBytes sql.NullInt64
	var firstArchivedAt, lastArchivedAt sql.NullTime

	err := r.db.QueryRow(query, tableName).Scan(
		&totalRecords,
		&totalOriginalBytes,
		&totalCompressedBytes,
		&totalStorageBytes,
		&firstArchivedAt,
		&lastArchivedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}

	stats["total_records"] = totalRecords
	if totalOriginalBytes.Valid {
		stats["total_original_bytes"] = totalOriginalBytes.Int64
	}
	if totalCompressedBytes.Valid {
		stats["total_compressed_bytes"] = totalCompressedBytes.Int64
	}
	if totalStorageBytes.Valid {
		stats["total_storage_bytes"] = totalStorageBytes.Int64
	}
	if firstArchivedAt.Valid {
		stats["first_archived_at"] = firstArchivedAt.Time
	}
	if lastArchivedAt.Valid {
		stats["last_archived_at"] = lastArchivedAt.Time
	}

	return stats, nil
}

// Count returns the total number of archived records
func (r *ArchivedRecordRepository) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM archived_records`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count archived records: %w", err)
	}

	return count, nil
}
