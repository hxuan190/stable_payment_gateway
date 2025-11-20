package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/lib/pq"
)

var (
	// ErrTransactionHashNotFound is returned when a transaction hash is not found
	ErrTransactionHashNotFound = errors.New("transaction hash not found")
	// ErrTransactionHashAlreadyExists is returned when a transaction hash already exists
	ErrTransactionHashAlreadyExists = errors.New("transaction hash already exists for this record")
)

// TransactionHashRepository handles database operations for transaction hashes
type TransactionHashRepository struct {
	db *sql.DB
}

// NewTransactionHashRepository creates a new transaction hash repository
func NewTransactionHashRepository(db *sql.DB) *TransactionHashRepository {
	return &TransactionHashRepository{
		db: db,
	}
}

// Create inserts a new transaction hash
func (r *TransactionHashRepository) Create(hash *model.TransactionHash) error {
	if hash == nil {
		return errors.New("transaction hash cannot be nil")
	}

	// Generate UUID if not provided
	if hash.ID == uuid.Nil {
		hash.ID = uuid.New()
	}

	query := `
		INSERT INTO transaction_hashes (
			id, table_name, record_id,
			data_hash, previous_hash,
			merkle_root, merkle_proof,
			block_number, block_date,
			record_snapshot,
			verified, verified_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if hash.CreatedAt.IsZero() {
		hash.CreatedAt = now
	}
	if hash.UpdatedAt.IsZero() {
		hash.UpdatedAt = now
	}

	err := r.db.QueryRow(
		query,
		hash.ID,
		hash.SourceTableName,
		hash.RecordID,
		hash.DataHash,
		hash.PreviousHash,
		hash.MerkleRoot,
		pq.Array(hash.MerkleProof),
		hash.BlockNumber,
		hash.BlockDate,
		hash.RecordSnapshot,
		hash.Verified,
		hash.VerifiedAt,
		hash.CreatedAt,
		hash.UpdatedAt,
	).Scan(&hash.ID, &hash.CreatedAt, &hash.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation
		if err.Error() == `pq: duplicate key value violates unique constraint "unique_transaction_hash"` {
			return ErrTransactionHashAlreadyExists
		}
		return fmt.Errorf("failed to create transaction hash: %w", err)
	}

	return nil
}

// GetByID retrieves a transaction hash by ID
func (r *TransactionHashRepository) GetByID(id uuid.UUID) (*model.TransactionHash, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid transaction hash ID")
	}

	query := `
		SELECT
			id, table_name, record_id,
			data_hash, previous_hash,
			merkle_root, merkle_proof,
			block_number, block_date,
			record_snapshot,
			verified, verified_at,
			created_at, updated_at
		FROM transaction_hashes
		WHERE id = $1
	`

	hash := &model.TransactionHash{}
	err := r.db.QueryRow(query, id).Scan(
		&hash.ID,
		&hash.SourceTableName,
		&hash.RecordID,
		&hash.DataHash,
		&hash.PreviousHash,
		&hash.MerkleRoot,
		pq.Array(&hash.MerkleProof),
		&hash.BlockNumber,
		&hash.BlockDate,
		&hash.RecordSnapshot,
		&hash.Verified,
		&hash.VerifiedAt,
		&hash.CreatedAt,
		&hash.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTransactionHashNotFound
		}
		return nil, fmt.Errorf("failed to get transaction hash by ID: %w", err)
	}

	return hash, nil
}

// GetByRecord retrieves a transaction hash by table name and record ID
func (r *TransactionHashRepository) GetByRecord(tableName string, recordID uuid.UUID) (*model.TransactionHash, error) {
	if tableName == "" {
		return nil, errors.New("table name cannot be empty")
	}
	if recordID == uuid.Nil {
		return nil, errors.New("record ID cannot be empty")
	}

	query := `
		SELECT
			id, table_name, record_id,
			data_hash, previous_hash,
			merkle_root, merkle_proof,
			block_number, block_date,
			record_snapshot,
			verified, verified_at,
			created_at, updated_at
		FROM transaction_hashes
		WHERE table_name = $1 AND record_id = $2
	`

	hash := &model.TransactionHash{}
	err := r.db.QueryRow(query, tableName, recordID).Scan(
		&hash.ID,
		&hash.SourceTableName,
		&hash.RecordID,
		&hash.DataHash,
		&hash.PreviousHash,
		&hash.MerkleRoot,
		pq.Array(&hash.MerkleProof),
		&hash.BlockNumber,
		&hash.BlockDate,
		&hash.RecordSnapshot,
		&hash.Verified,
		&hash.VerifiedAt,
		&hash.CreatedAt,
		&hash.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTransactionHashNotFound
		}
		return nil, fmt.Errorf("failed to get transaction hash by record: %w", err)
	}

	return hash, nil
}

// ListByTableName retrieves transaction hashes by table name with pagination
func (r *TransactionHashRepository) ListByTableName(tableName string, limit, offset int) ([]*model.TransactionHash, error) {
	if tableName == "" {
		return nil, errors.New("table name cannot be empty")
	}

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, table_name, record_id,
			data_hash, previous_hash,
			merkle_root, merkle_proof,
			block_number, block_date,
			record_snapshot,
			verified, verified_at,
			created_at, updated_at
		FROM transaction_hashes
		WHERE table_name = $1
		ORDER BY block_date DESC, created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, tableName, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transaction hashes by table name: %w", err)
	}
	defer rows.Close()

	hashes := make([]*model.TransactionHash, 0)
	for rows.Next() {
		hash := &model.TransactionHash{}
		err := rows.Scan(
			&hash.ID,
			&hash.SourceTableName,
			&hash.RecordID,
			&hash.DataHash,
			&hash.PreviousHash,
			&hash.MerkleRoot,
			pq.Array(&hash.MerkleProof),
			&hash.BlockNumber,
			&hash.BlockDate,
			&hash.RecordSnapshot,
			&hash.Verified,
			&hash.VerifiedAt,
			&hash.CreatedAt,
			&hash.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction hash: %w", err)
		}
		hashes = append(hashes, hash)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction hashes: %w", err)
	}

	return hashes, nil
}

// ListByBlockDate retrieves transaction hashes by block date
func (r *TransactionHashRepository) ListByBlockDate(tableName string, blockDate time.Time) ([]*model.TransactionHash, error) {
	if tableName == "" {
		return nil, errors.New("table name cannot be empty")
	}

	query := `
		SELECT
			id, table_name, record_id,
			data_hash, previous_hash,
			merkle_root, merkle_proof,
			block_number, block_date,
			record_snapshot,
			verified, verified_at,
			created_at, updated_at
		FROM transaction_hashes
		WHERE table_name = $1 AND block_date = $2
		ORDER BY block_number ASC, created_at ASC
	`

	rows, err := r.db.Query(query, tableName, blockDate)
	if err != nil {
		return nil, fmt.Errorf("failed to list transaction hashes by block date: %w", err)
	}
	defer rows.Close()

	hashes := make([]*model.TransactionHash, 0)
	for rows.Next() {
		hash := &model.TransactionHash{}
		err := rows.Scan(
			&hash.ID,
			&hash.SourceTableName,
			&hash.RecordID,
			&hash.DataHash,
			&hash.PreviousHash,
			&hash.MerkleRoot,
			pq.Array(&hash.MerkleProof),
			&hash.BlockNumber,
			&hash.BlockDate,
			&hash.RecordSnapshot,
			&hash.Verified,
			&hash.VerifiedAt,
			&hash.CreatedAt,
			&hash.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction hash: %w", err)
		}
		hashes = append(hashes, hash)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction hashes: %w", err)
	}

	return hashes, nil
}

// Update updates an existing transaction hash
func (r *TransactionHashRepository) Update(hash *model.TransactionHash) error {
	if hash == nil {
		return errors.New("transaction hash cannot be nil")
	}
	if hash.ID == uuid.Nil {
		return errors.New("invalid transaction hash ID")
	}

	query := `
		UPDATE transaction_hashes SET
			merkle_root = $2,
			merkle_proof = $3,
			verified = $4,
			verified_at = $5,
			updated_at = $6
		WHERE id = $1
		RETURNING updated_at
	`

	hash.UpdatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		hash.ID,
		hash.MerkleRoot,
		pq.Array(hash.MerkleProof),
		hash.Verified,
		hash.VerifiedAt,
		hash.UpdatedAt,
	).Scan(&hash.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrTransactionHashNotFound
		}
		return fmt.Errorf("failed to update transaction hash: %w", err)
	}

	return nil
}

// MarkAsVerified marks a transaction hash as verified
func (r *TransactionHashRepository) MarkAsVerified(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid transaction hash ID")
	}

	query := `
		UPDATE transaction_hashes
		SET
			verified = TRUE,
			verified_at = $2,
			updated_at = $2
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(query, id, sql.NullTime{Time: now, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to mark transaction hash as verified: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTransactionHashNotFound
	}

	return nil
}

// UpdateMerkleRoot updates the Merkle root and proof for a batch of hashes
func (r *TransactionHashRepository) UpdateMerkleRoot(tableName string, blockDate time.Time, merkleRoot string) error {
	if tableName == "" {
		return errors.New("table name cannot be empty")
	}
	if merkleRoot == "" {
		return errors.New("merkle root cannot be empty")
	}

	query := `
		UPDATE transaction_hashes
		SET
			merkle_root = $3,
			updated_at = $4
		WHERE table_name = $1 AND block_date = $2
	`

	result, err := r.db.Exec(query, tableName, blockDate, sql.NullString{String: merkleRoot, Valid: true}, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update merkle root: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("no transaction hashes found for the given table and block date")
	}

	return nil
}

// ListUnverified retrieves unverified transaction hashes for a table
func (r *TransactionHashRepository) ListUnverified(tableName string, limit int) ([]*model.TransactionHash, error) {
	if tableName == "" {
		return nil, errors.New("table name cannot be empty")
	}

	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT
			id, table_name, record_id,
			data_hash, previous_hash,
			merkle_root, merkle_proof,
			block_number, block_date,
			record_snapshot,
			verified, verified_at,
			created_at, updated_at
		FROM transaction_hashes
		WHERE table_name = $1 AND verified = FALSE
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := r.db.Query(query, tableName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list unverified transaction hashes: %w", err)
	}
	defer rows.Close()

	hashes := make([]*model.TransactionHash, 0)
	for rows.Next() {
		hash := &model.TransactionHash{}
		err := rows.Scan(
			&hash.ID,
			&hash.SourceTableName,
			&hash.RecordID,
			&hash.DataHash,
			&hash.PreviousHash,
			&hash.MerkleRoot,
			pq.Array(&hash.MerkleProof),
			&hash.BlockNumber,
			&hash.BlockDate,
			&hash.RecordSnapshot,
			&hash.Verified,
			&hash.VerifiedAt,
			&hash.CreatedAt,
			&hash.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction hash: %w", err)
		}
		hashes = append(hashes, hash)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction hashes: %w", err)
	}

	return hashes, nil
}

// GetLatestByTable retrieves the latest transaction hash for a table
func (r *TransactionHashRepository) GetLatestByTable(tableName string) (*model.TransactionHash, error) {
	if tableName == "" {
		return nil, errors.New("table name cannot be empty")
	}

	query := `
		SELECT
			id, table_name, record_id,
			data_hash, previous_hash,
			merkle_root, merkle_proof,
			block_number, block_date,
			record_snapshot,
			verified, verified_at,
			created_at, updated_at
		FROM transaction_hashes
		WHERE table_name = $1
		ORDER BY block_date DESC, created_at DESC
		LIMIT 1
	`

	hash := &model.TransactionHash{}
	err := r.db.QueryRow(query, tableName).Scan(
		&hash.ID,
		&hash.SourceTableName,
		&hash.RecordID,
		&hash.DataHash,
		&hash.PreviousHash,
		&hash.MerkleRoot,
		pq.Array(&hash.MerkleProof),
		&hash.BlockNumber,
		&hash.BlockDate,
		&hash.RecordSnapshot,
		&hash.Verified,
		&hash.VerifiedAt,
		&hash.CreatedAt,
		&hash.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTransactionHashNotFound
		}
		return nil, fmt.Errorf("failed to get latest transaction hash: %w", err)
	}

	return hash, nil
}

// Count returns the total number of transaction hashes
func (r *TransactionHashRepository) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM transaction_hashes`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count transaction hashes: %w", err)
	}

	return count, nil
}

// CountByTableAndDate returns the count of transaction hashes for a table and block date
func (r *TransactionHashRepository) CountByTableAndDate(tableName string, blockDate time.Time) (int64, error) {
	if tableName == "" {
		return 0, errors.New("table name cannot be empty")
	}

	query := `
		SELECT COUNT(*)
		FROM transaction_hashes
		WHERE table_name = $1 AND block_date = $2
	`

	var count int64
	err := r.db.QueryRow(query, tableName, blockDate).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count transaction hashes by table and date: %w", err)
	}

	return count, nil
}
