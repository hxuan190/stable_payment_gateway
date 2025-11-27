package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"gorm.io/gorm"
)

var (
	// ErrArchivedRecordNotFound is returned when an archived record is not found
	ErrArchivedRecordNotFound = errors.New("archived record not found")
)

// ArchivedRecordRepository handles database operations for archived records
type ArchivedRecordRepository struct {
	db *gorm.DB
}

// NewArchivedRecordRepository creates a new archived record repository
func NewArchivedRecordRepository(db *gorm.DB) *ArchivedRecordRepository {
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

	if err := r.db.Create(record).Error; err != nil {
		return fmt.Errorf("failed to create archived record: %w", err)
	}

	return nil
}

// GetByID retrieves an archived record by ID
func (r *ArchivedRecordRepository) GetByID(id uuid.UUID) (*model.ArchivedRecord, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid archived record ID")
	}

	record := &model.ArchivedRecord{}
	if err := r.db.Where("id = ?", id).First(record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArchivedRecordNotFound
		}
		return nil, err
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

	record := &model.ArchivedRecord{}
	if err := r.db.Where("table_name = ? AND original_id = ?", tableName, originalID).First(record).Error; err != nil {
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrArchivedRecordNotFound
			}
			return nil, err
		}

		return record, nil
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

	records := make([]*model.ArchivedRecord, 0)
	if err := r.db.Where("table_name = ?", tableName).Order("archived_at DESC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, err
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

	if err := r.db.Save(record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

	record, err := r.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get archived record: %w", err)
	}

	record.RestoreInitiatedAt = sql.NullTime{Time: time.Now(), Valid: true}
	record.RestoreTier = sql.NullString{String: string(tier), Valid: true}
	record.RestoreInitiatedBy = sql.NullString{String: initiatedBy, Valid: true}
	record.UpdatedAt = time.Now()

	return r.Update(record)
}

// MarkAsRestored marks an archived record as restored
func (r *ArchivedRecordRepository) MarkAsRestored(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid archived record ID")
	}

	record, err := r.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get archived record: %w", err)
	}

	record.RestoredAt = sql.NullTime{Time: time.Now(), Valid: true}
	record.UpdatedAt = time.Now()

	if err := r.db.Save(record).Error; err != nil {
		return fmt.Errorf("failed to mark as restored: %w", err)
	}
	return r.Update(record)
}

// Count returns the total number of archived records
func (r *ArchivedRecordRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&model.ArchivedRecord{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count archived records: %w", err)
	}
	return count, nil
}
