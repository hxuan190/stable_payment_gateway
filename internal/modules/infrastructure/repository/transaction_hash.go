package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/infrastructure/domain"
	"gorm.io/gorm"
)

var (
	ErrTransactionHashNotFound      = errors.New("transaction hash not found")
	ErrTransactionHashAlreadyExists = errors.New("transaction hash already exists for this record")
)

type TransactionHashRepository struct {
	db *gorm.DB
}

func NewTransactionHashRepository(db *gorm.DB) *TransactionHashRepository {
	return &TransactionHashRepository{
		db: db,
	}
}

func (r *TransactionHashRepository) Create(hash *domain.TransactionHash) error {
	if hash == nil {
		return errors.New("transaction hash cannot be nil")
	}

	if hash.ID == uuid.Nil {
		hash.ID = uuid.New()
	}

	if err := r.db.Create(hash).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrTransactionHashAlreadyExists
		}
		return fmt.Errorf("failed to create transaction hash: %w", err)
	}

	return nil
}

func (r *TransactionHashRepository) GetByID(id uuid.UUID) (*domain.TransactionHash, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid transaction hash ID")
	}

	hash := &domain.TransactionHash{}
	if err := r.db.Where("id = ?", id).First(hash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTransactionHashNotFound
		}
		return nil, fmt.Errorf("failed to get transaction hash by ID: %w", err)
	}

	return hash, nil
}

func (r *TransactionHashRepository) GetByRecord(tableName string, recordID uuid.UUID) (*domain.TransactionHash, error) {
	if tableName == "" {
		return nil, errors.New("table name cannot be empty")
	}
	if recordID == uuid.Nil {
		return nil, errors.New("record ID cannot be empty")
	}

	hash := &domain.TransactionHash{}
	if err := r.db.Where("table_name = ? AND record_id = ?", tableName, recordID).First(hash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTransactionHashNotFound
		}
		return nil, fmt.Errorf("failed to get transaction hash by record: %w", err)
	}

	return hash, nil
}

func (r *TransactionHashRepository) ListByTableName(tableName string, limit, offset int) ([]*domain.TransactionHash, error) {
	if tableName == "" {
		return nil, errors.New("table name cannot be empty")
	}

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var hashes []*domain.TransactionHash
	if err := r.db.Where("table_name = ?", tableName).
		Order("block_date DESC, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&hashes).Error; err != nil {
		return nil, fmt.Errorf("failed to list transaction hashes by table name: %w", err)
	}

	return hashes, nil
}

func (r *TransactionHashRepository) ListByBlockDate(tableName string, blockDate time.Time) ([]*domain.TransactionHash, error) {
	if tableName == "" {
		return nil, errors.New("table name cannot be empty")
	}

	var hashes []*domain.TransactionHash
	if err := r.db.Where("table_name = ? AND block_date = ?", tableName, blockDate).
		Order("block_number ASC, created_at ASC").
		Find(&hashes).Error; err != nil {
		return nil, fmt.Errorf("failed to list transaction hashes by block date: %w", err)
	}

	return hashes, nil
}

func (r *TransactionHashRepository) Update(hash *domain.TransactionHash) error {
	if hash == nil {
		return errors.New("transaction hash cannot be nil")
	}
	if hash.ID == uuid.Nil {
		return errors.New("invalid transaction hash ID")
	}

	hash.UpdatedAt = time.Now()

	result := r.db.Model(&domain.TransactionHash{}).Where("id = ?", hash.ID).Updates(map[string]interface{}{
		"merkle_root":  hash.MerkleRoot,
		"merkle_proof": hash.MerkleProof,
		"verified":     hash.Verified,
		"verified_at":  hash.VerifiedAt,
		"updated_at":   hash.UpdatedAt,
	})

	if result.Error != nil {
		return fmt.Errorf("failed to update transaction hash: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrTransactionHashNotFound
	}

	return nil
}

func (r *TransactionHashRepository) MarkAsVerified(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid transaction hash ID")
	}

	now := time.Now()
	result := r.db.Model(&domain.TransactionHash{}).Where("id = ?", id).Updates(map[string]interface{}{
		"verified":    true,
		"verified_at": sql.NullTime{Time: now, Valid: true},
		"updated_at":  now,
	})

	if result.Error != nil {
		return fmt.Errorf("failed to mark transaction hash as verified: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrTransactionHashNotFound
	}

	return nil
}

func (r *TransactionHashRepository) UpdateMerkleRoot(tableName string, blockDate time.Time, merkleRoot string) error {
	if tableName == "" {
		return errors.New("table name cannot be empty")
	}
	if merkleRoot == "" {
		return errors.New("merkle root cannot be empty")
	}

	result := r.db.Model(&domain.TransactionHash{}).
		Where("table_name = ? AND block_date = ?", tableName, blockDate).
		Updates(map[string]interface{}{
			"merkle_root": sql.NullString{String: merkleRoot, Valid: true},
			"updated_at":  time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update merkle root: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("no transaction hashes found for the given table and block date")
	}

	return nil
}

func (r *TransactionHashRepository) ListUnverified(tableName string, limit int) ([]*domain.TransactionHash, error) {
	if tableName == "" {
		return nil, errors.New("table name cannot be empty")
	}

	if limit <= 0 {
		limit = 100
	}

	var hashes []*domain.TransactionHash
	if err := r.db.Where("table_name = ? AND verified = ?", tableName, false).
		Order("created_at ASC").
		Limit(limit).
		Find(&hashes).Error; err != nil {
		return nil, fmt.Errorf("failed to list unverified transaction hashes: %w", err)
	}

	return hashes, nil
}

func (r *TransactionHashRepository) GetLatestByTable(tableName string) (*domain.TransactionHash, error) {
	if tableName == "" {
		return nil, errors.New("table name cannot be empty")
	}

	hash := &domain.TransactionHash{}
	if err := r.db.Where("table_name = ?", tableName).
		Order("block_date DESC, created_at DESC").
		First(hash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTransactionHashNotFound
		}
		return nil, fmt.Errorf("failed to get latest transaction hash: %w", err)
	}

	return hash, nil
}

func (r *TransactionHashRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&domain.TransactionHash{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count transaction hashes: %w", err)
	}

	return count, nil
}

func (r *TransactionHashRepository) CountByTableAndDate(tableName string, blockDate time.Time) (int64, error) {
	if tableName == "" {
		return 0, errors.New("table name cannot be empty")
	}

	var count int64
	if err := r.db.Model(&domain.TransactionHash{}).
		Where("table_name = ? AND block_date = ?", tableName, blockDate).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count transaction hashes by table and date: %w", err)
	}

	return count, nil
}
