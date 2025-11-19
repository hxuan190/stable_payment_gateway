package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/shopspring/decimal"
)

var (
	// ErrWalletMappingNotFound is returned when a wallet mapping is not found
	ErrWalletMappingNotFound = errors.New("wallet identity mapping not found")
	// ErrWalletMappingAlreadyExists is returned when a wallet mapping already exists
	ErrWalletMappingAlreadyExists = errors.New("wallet identity mapping already exists")
)

// WalletIdentityMappingRepository handles database operations for wallet identity mappings
type WalletIdentityMappingRepository struct {
	db *sql.DB
}

// NewWalletIdentityMappingRepository creates a new wallet identity mapping repository
func NewWalletIdentityMappingRepository(db *sql.DB) *WalletIdentityMappingRepository {
	return &WalletIdentityMappingRepository{
		db: db,
	}
}

// Create inserts a new wallet identity mapping
func (r *WalletIdentityMappingRepository) Create(mapping *model.WalletIdentityMapping) error {
	if mapping == nil {
		return errors.New("wallet identity mapping cannot be nil")
	}

	// Generate UUID if not provided
	if mapping.ID == uuid.Nil {
		mapping.ID = uuid.New()
	}

	// Compute wallet hash if not provided
	if mapping.WalletAddressHash == "" {
		mapping.WalletAddressHash = model.ComputeWalletHash(mapping.WalletAddress)
	}

	query := `
		INSERT INTO wallet_identity_mappings (
			id, wallet_address, blockchain, wallet_address_hash,
			user_id, kyc_status,
			first_seen_at, last_seen_at,
			payment_count, total_volume_usd,
			is_flagged, flagged_at, flagged_reason,
			metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if mapping.CreatedAt.IsZero() {
		mapping.CreatedAt = now
	}
	if mapping.UpdatedAt.IsZero() {
		mapping.UpdatedAt = now
	}
	if mapping.FirstSeenAt.IsZero() {
		mapping.FirstSeenAt = now
	}
	if mapping.LastSeenAt.IsZero() {
		mapping.LastSeenAt = now
	}

	err := r.db.QueryRow(
		query,
		mapping.ID,
		mapping.WalletAddress,
		mapping.Blockchain,
		mapping.WalletAddressHash,
		mapping.UserID,
		mapping.KYCStatus,
		mapping.FirstSeenAt,
		mapping.LastSeenAt,
		mapping.PaymentCount,
		mapping.TotalVolumeUSD,
		mapping.IsFlagged,
		mapping.FlaggedAt,
		mapping.FlaggedReason,
		mapping.Metadata,
		mapping.CreatedAt,
		mapping.UpdatedAt,
	).Scan(&mapping.ID, &mapping.CreatedAt, &mapping.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation
		if err.Error() == `pq: duplicate key value violates unique constraint "unique_wallet_blockchain"` {
			return ErrWalletMappingAlreadyExists
		}
		return fmt.Errorf("failed to create wallet identity mapping: %w", err)
	}

	return nil
}

// GetByID retrieves a wallet identity mapping by ID
func (r *WalletIdentityMappingRepository) GetByID(id uuid.UUID) (*model.WalletIdentityMapping, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid wallet mapping ID")
	}

	query := `
		SELECT
			id, wallet_address, blockchain, wallet_address_hash,
			user_id, kyc_status,
			first_seen_at, last_seen_at,
			payment_count, total_volume_usd,
			is_flagged, flagged_at, flagged_reason,
			metadata, created_at, updated_at
		FROM wallet_identity_mappings
		WHERE id = $1
	`

	mapping := &model.WalletIdentityMapping{}
	err := r.db.QueryRow(query, id).Scan(
		&mapping.ID,
		&mapping.WalletAddress,
		&mapping.Blockchain,
		&mapping.WalletAddressHash,
		&mapping.UserID,
		&mapping.KYCStatus,
		&mapping.FirstSeenAt,
		&mapping.LastSeenAt,
		&mapping.PaymentCount,
		&mapping.TotalVolumeUSD,
		&mapping.IsFlagged,
		&mapping.FlaggedAt,
		&mapping.FlaggedReason,
		&mapping.Metadata,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrWalletMappingNotFound
		}
		return nil, fmt.Errorf("failed to get wallet identity mapping by ID: %w", err)
	}

	return mapping, nil
}

// GetByWalletAddress retrieves a wallet identity mapping by wallet address and blockchain
func (r *WalletIdentityMappingRepository) GetByWalletAddress(walletAddress string, blockchain model.Chain) (*model.WalletIdentityMapping, error) {
	if walletAddress == "" {
		return nil, errors.New("wallet address cannot be empty")
	}

	query := `
		SELECT
			id, wallet_address, blockchain, wallet_address_hash,
			user_id, kyc_status,
			first_seen_at, last_seen_at,
			payment_count, total_volume_usd,
			is_flagged, flagged_at, flagged_reason,
			metadata, created_at, updated_at
		FROM wallet_identity_mappings
		WHERE wallet_address = $1 AND blockchain = $2
	`

	mapping := &model.WalletIdentityMapping{}
	err := r.db.QueryRow(query, walletAddress, blockchain).Scan(
		&mapping.ID,
		&mapping.WalletAddress,
		&mapping.Blockchain,
		&mapping.WalletAddressHash,
		&mapping.UserID,
		&mapping.KYCStatus,
		&mapping.FirstSeenAt,
		&mapping.LastSeenAt,
		&mapping.PaymentCount,
		&mapping.TotalVolumeUSD,
		&mapping.IsFlagged,
		&mapping.FlaggedAt,
		&mapping.FlaggedReason,
		&mapping.Metadata,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrWalletMappingNotFound
		}
		return nil, fmt.Errorf("failed to get wallet identity mapping by address: %w", err)
	}

	return mapping, nil
}

// GetByUserID retrieves all wallet identity mappings for a user
func (r *WalletIdentityMappingRepository) GetByUserID(userID uuid.UUID) ([]*model.WalletIdentityMapping, error) {
	if userID == uuid.Nil {
		return nil, errors.New("user ID cannot be empty")
	}

	query := `
		SELECT
			id, wallet_address, blockchain, wallet_address_hash,
			user_id, kyc_status,
			first_seen_at, last_seen_at,
			payment_count, total_volume_usd,
			is_flagged, flagged_at, flagged_reason,
			metadata, created_at, updated_at
		FROM wallet_identity_mappings
		WHERE user_id = $1
		ORDER BY last_seen_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet identity mappings by user ID: %w", err)
	}
	defer rows.Close()

	mappings := make([]*model.WalletIdentityMapping, 0)
	for rows.Next() {
		mapping := &model.WalletIdentityMapping{}
		err := rows.Scan(
			&mapping.ID,
			&mapping.WalletAddress,
			&mapping.Blockchain,
			&mapping.WalletAddressHash,
			&mapping.UserID,
			&mapping.KYCStatus,
			&mapping.FirstSeenAt,
			&mapping.LastSeenAt,
			&mapping.PaymentCount,
			&mapping.TotalVolumeUSD,
			&mapping.IsFlagged,
			&mapping.FlaggedAt,
			&mapping.FlaggedReason,
			&mapping.Metadata,
			&mapping.CreatedAt,
			&mapping.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wallet identity mapping: %w", err)
		}
		mappings = append(mappings, mapping)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating wallet identity mappings: %w", err)
	}

	return mappings, nil
}

// Update updates an existing wallet identity mapping
func (r *WalletIdentityMappingRepository) Update(mapping *model.WalletIdentityMapping) error {
	if mapping == nil {
		return errors.New("wallet identity mapping cannot be nil")
	}
	if mapping.ID == uuid.Nil {
		return errors.New("invalid wallet mapping ID")
	}

	// Update wallet hash
	mapping.WalletAddressHash = model.ComputeWalletHash(mapping.WalletAddress)

	query := `
		UPDATE wallet_identity_mappings SET
			wallet_address = $2,
			blockchain = $3,
			wallet_address_hash = $4,
			user_id = $5,
			kyc_status = $6,
			first_seen_at = $7,
			last_seen_at = $8,
			payment_count = $9,
			total_volume_usd = $10,
			is_flagged = $11,
			flagged_at = $12,
			flagged_reason = $13,
			metadata = $14,
			updated_at = $15
		WHERE id = $1
		RETURNING updated_at
	`

	mapping.UpdatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		mapping.ID,
		mapping.WalletAddress,
		mapping.Blockchain,
		mapping.WalletAddressHash,
		mapping.UserID,
		mapping.KYCStatus,
		mapping.FirstSeenAt,
		mapping.LastSeenAt,
		mapping.PaymentCount,
		mapping.TotalVolumeUSD,
		mapping.IsFlagged,
		mapping.FlaggedAt,
		mapping.FlaggedReason,
		mapping.Metadata,
		mapping.UpdatedAt,
	).Scan(&mapping.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrWalletMappingNotFound
		}
		return fmt.Errorf("failed to update wallet identity mapping: %w", err)
	}

	return nil
}

// UpdateActivity updates the payment activity for a wallet mapping
func (r *WalletIdentityMappingRepository) UpdateActivity(id uuid.UUID, amountUSD decimal.Decimal) error {
	if id == uuid.Nil {
		return errors.New("invalid wallet mapping ID")
	}

	query := `
		UPDATE wallet_identity_mappings
		SET
			payment_count = payment_count + 1,
			total_volume_usd = total_volume_usd + $2,
			last_seen_at = $3,
			updated_at = $3
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(query, id, amountUSD, now)
	if err != nil {
		return fmt.Errorf("failed to update wallet activity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrWalletMappingNotFound
	}

	return nil
}

// Flag flags a wallet mapping as suspicious
func (r *WalletIdentityMappingRepository) Flag(id uuid.UUID, reason string) error {
	if id == uuid.Nil {
		return errors.New("invalid wallet mapping ID")
	}
	if reason == "" {
		return errors.New("flag reason cannot be empty")
	}

	query := `
		UPDATE wallet_identity_mappings
		SET
			is_flagged = TRUE,
			flagged_at = $2,
			flagged_reason = $3,
			updated_at = $2
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(query, id, sql.NullTime{Time: now, Valid: true}, sql.NullString{String: reason, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to flag wallet: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrWalletMappingNotFound
	}

	return nil
}

// Unflag removes the flag from a wallet mapping
func (r *WalletIdentityMappingRepository) Unflag(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid wallet mapping ID")
	}

	query := `
		UPDATE wallet_identity_mappings
		SET
			is_flagged = FALSE,
			flagged_at = NULL,
			flagged_reason = NULL,
			updated_at = $2
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to unflag wallet: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrWalletMappingNotFound
	}

	return nil
}

// ListActive retrieves active wallet mappings (seen in last 7 days)
func (r *WalletIdentityMappingRepository) ListActive(limit, offset int) ([]*model.WalletIdentityMapping, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, wallet_address, blockchain, wallet_address_hash,
			user_id, kyc_status,
			first_seen_at, last_seen_at,
			payment_count, total_volume_usd,
			is_flagged, flagged_at, flagged_reason,
			metadata, created_at, updated_at
		FROM wallet_identity_mappings
		WHERE last_seen_at >= CURRENT_TIMESTAMP - INTERVAL '7 days'
		ORDER BY last_seen_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list active wallet mappings: %w", err)
	}
	defer rows.Close()

	mappings := make([]*model.WalletIdentityMapping, 0)
	for rows.Next() {
		mapping := &model.WalletIdentityMapping{}
		err := rows.Scan(
			&mapping.ID,
			&mapping.WalletAddress,
			&mapping.Blockchain,
			&mapping.WalletAddressHash,
			&mapping.UserID,
			&mapping.KYCStatus,
			&mapping.FirstSeenAt,
			&mapping.LastSeenAt,
			&mapping.PaymentCount,
			&mapping.TotalVolumeUSD,
			&mapping.IsFlagged,
			&mapping.FlaggedAt,
			&mapping.FlaggedReason,
			&mapping.Metadata,
			&mapping.CreatedAt,
			&mapping.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wallet identity mapping: %w", err)
		}
		mappings = append(mappings, mapping)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating wallet identity mappings: %w", err)
	}

	return mappings, nil
}

// ListFlagged retrieves all flagged wallet mappings
func (r *WalletIdentityMappingRepository) ListFlagged(limit, offset int) ([]*model.WalletIdentityMapping, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, wallet_address, blockchain, wallet_address_hash,
			user_id, kyc_status,
			first_seen_at, last_seen_at,
			payment_count, total_volume_usd,
			is_flagged, flagged_at, flagged_reason,
			metadata, created_at, updated_at
		FROM wallet_identity_mappings
		WHERE is_flagged = TRUE
		ORDER BY flagged_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list flagged wallet mappings: %w", err)
	}
	defer rows.Close()

	mappings := make([]*model.WalletIdentityMapping, 0)
	for rows.Next() {
		mapping := &model.WalletIdentityMapping{}
		err := rows.Scan(
			&mapping.ID,
			&mapping.WalletAddress,
			&mapping.Blockchain,
			&mapping.WalletAddressHash,
			&mapping.UserID,
			&mapping.KYCStatus,
			&mapping.FirstSeenAt,
			&mapping.LastSeenAt,
			&mapping.PaymentCount,
			&mapping.TotalVolumeUSD,
			&mapping.IsFlagged,
			&mapping.FlaggedAt,
			&mapping.FlaggedReason,
			&mapping.Metadata,
			&mapping.CreatedAt,
			&mapping.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wallet identity mapping: %w", err)
		}
		mappings = append(mappings, mapping)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating wallet identity mappings: %w", err)
	}

	return mappings, nil
}

// Count returns the total number of wallet identity mappings
func (r *WalletIdentityMappingRepository) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM wallet_identity_mappings`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count wallet identity mappings: %w", err)
	}

	return count, nil
}
