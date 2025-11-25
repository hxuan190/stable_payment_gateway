package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

var (
	// ErrTreasuryWalletNotFound is returned when a treasury wallet is not found
	ErrTreasuryWalletNotFound = errors.New("treasury wallet not found")
	// ErrTreasuryWalletAlreadyExists is returned when a treasury wallet already exists
	ErrTreasuryWalletAlreadyExists = errors.New("treasury wallet already exists for this blockchain and address")
)

// TreasuryWalletRepository handles database operations for treasury wallets
type TreasuryWalletRepository struct {
	db *sql.DB
}

// NewTreasuryWalletRepository creates a new treasury wallet repository
func NewTreasuryWalletRepository(db *sql.DB) *TreasuryWalletRepository {
	return &TreasuryWalletRepository{
		db: db,
	}
}

// Create inserts a new treasury wallet
func (r *TreasuryWalletRepository) Create(wallet *model.TreasuryWallet) error {
	if wallet == nil {
		return errors.New("treasury wallet cannot be nil")
	}

	// Generate UUID if not provided
	if wallet.ID == uuid.Nil {
		wallet.ID = uuid.New()
	}

	// Compute address hash if not provided
	if wallet.AddressHash == "" {
		wallet.AddressHash = model.ComputeAddressHash(wallet.Address)
	}

	query := `
		INSERT INTO treasury_wallets (
			id, wallet_type, blockchain, address, address_hash,
			is_multisig, multisig_scheme, multisig_signers, multisig_threshold,
			balance_crypto, balance_usd, balance_last_updated_at,
			sweep_threshold_usd, sweep_target_wallet_id, sweep_buffer_usd, auto_sweep_enabled,
			last_swept_at, last_deposit_at, last_withdrawal_at, total_swept_amount_usd,
			status, is_monitored,
			encrypted_private_key, encryption_method, key_storage_location,
			label, description, metadata,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32
		)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if wallet.CreatedAt.IsZero() {
		wallet.CreatedAt = now
	}
	if wallet.UpdatedAt.IsZero() {
		wallet.UpdatedAt = now
	}

	err := r.db.QueryRow(
		query,
		wallet.ID,
		wallet.WalletType,
		wallet.Blockchain,
		wallet.Address,
		wallet.AddressHash,
		wallet.IsMultisig,
		wallet.MultisigScheme,
		pq.Array(wallet.MultisigSigners),
		wallet.MultisigThreshold,
		wallet.BalanceCrypto,
		wallet.BalanceUSD,
		wallet.BalanceLastUpdatedAt,
		wallet.SweepThresholdUSD,
		wallet.SweepTargetWalletID,
		wallet.SweepBufferUSD,
		wallet.AutoSweepEnabled,
		wallet.LastSweptAt,
		wallet.LastDepositAt,
		wallet.LastWithdrawalAt,
		wallet.TotalSweptAmountUSD,
		wallet.Status,
		wallet.IsMonitored,
		wallet.EncryptedPrivateKey,
		wallet.EncryptionMethod,
		wallet.KeyStorageLocation,
		wallet.Label,
		wallet.Description,
		wallet.Metadata,
		wallet.CreatedAt,
		wallet.UpdatedAt,
		wallet.CreatedBy,
		wallet.UpdatedBy,
	).Scan(&wallet.ID, &wallet.CreatedAt, &wallet.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation
		if err.Error() == `pq: duplicate key value violates unique constraint "idx_blockchain_address"` {
			return ErrTreasuryWalletAlreadyExists
		}
		return fmt.Errorf("failed to create treasury wallet: %w", err)
	}

	return nil
}

// GetByID retrieves a treasury wallet by ID
func (r *TreasuryWalletRepository) GetByID(id uuid.UUID) (*model.TreasuryWallet, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid treasury wallet ID")
	}

	query := `
		SELECT
			id, wallet_type, blockchain, address, address_hash,
			is_multisig, multisig_scheme, multisig_signers, multisig_threshold,
			balance_crypto, balance_usd, balance_last_updated_at,
			sweep_threshold_usd, sweep_target_wallet_id, sweep_buffer_usd, auto_sweep_enabled,
			last_swept_at, last_deposit_at, last_withdrawal_at, total_swept_amount_usd,
			status, is_monitored,
			encrypted_private_key, encryption_method, key_storage_location,
			label, description, metadata,
			created_at, updated_at, created_by, updated_by
		FROM treasury_wallets
		WHERE id = $1
	`

	wallet := &model.TreasuryWallet{}
	err := r.db.QueryRow(query, id).Scan(
		&wallet.ID,
		&wallet.WalletType,
		&wallet.Blockchain,
		&wallet.Address,
		&wallet.AddressHash,
		&wallet.IsMultisig,
		&wallet.MultisigScheme,
		pq.Array(&wallet.MultisigSigners),
		&wallet.MultisigThreshold,
		&wallet.BalanceCrypto,
		&wallet.BalanceUSD,
		&wallet.BalanceLastUpdatedAt,
		&wallet.SweepThresholdUSD,
		&wallet.SweepTargetWalletID,
		&wallet.SweepBufferUSD,
		&wallet.AutoSweepEnabled,
		&wallet.LastSweptAt,
		&wallet.LastDepositAt,
		&wallet.LastWithdrawalAt,
		&wallet.TotalSweptAmountUSD,
		&wallet.Status,
		&wallet.IsMonitored,
		&wallet.EncryptedPrivateKey,
		&wallet.EncryptionMethod,
		&wallet.KeyStorageLocation,
		&wallet.Label,
		&wallet.Description,
		&wallet.Metadata,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
		&wallet.CreatedBy,
		&wallet.UpdatedBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTreasuryWalletNotFound
		}
		return nil, fmt.Errorf("failed to get treasury wallet by ID: %w", err)
	}

	return wallet, nil
}

// GetByAddress retrieves a treasury wallet by blockchain and address
func (r *TreasuryWalletRepository) GetByAddress(blockchain model.Chain, address string) (*model.TreasuryWallet, error) {
	if address == "" {
		return nil, errors.New("address cannot be empty")
	}

	query := `
		SELECT
			id, wallet_type, blockchain, address, address_hash,
			is_multisig, multisig_scheme, multisig_signers, multisig_threshold,
			balance_crypto, balance_usd, balance_last_updated_at,
			sweep_threshold_usd, sweep_target_wallet_id, sweep_buffer_usd, auto_sweep_enabled,
			last_swept_at, last_deposit_at, last_withdrawal_at, total_swept_amount_usd,
			status, is_monitored,
			encrypted_private_key, encryption_method, key_storage_location,
			label, description, metadata,
			created_at, updated_at, created_by, updated_by
		FROM treasury_wallets
		WHERE blockchain = $1 AND address = $2
	`

	wallet := &model.TreasuryWallet{}
	err := r.db.QueryRow(query, blockchain, address).Scan(
		&wallet.ID,
		&wallet.WalletType,
		&wallet.Blockchain,
		&wallet.Address,
		&wallet.AddressHash,
		&wallet.IsMultisig,
		&wallet.MultisigScheme,
		pq.Array(&wallet.MultisigSigners),
		&wallet.MultisigThreshold,
		&wallet.BalanceCrypto,
		&wallet.BalanceUSD,
		&wallet.BalanceLastUpdatedAt,
		&wallet.SweepThresholdUSD,
		&wallet.SweepTargetWalletID,
		&wallet.SweepBufferUSD,
		&wallet.AutoSweepEnabled,
		&wallet.LastSweptAt,
		&wallet.LastDepositAt,
		&wallet.LastWithdrawalAt,
		&wallet.TotalSweptAmountUSD,
		&wallet.Status,
		&wallet.IsMonitored,
		&wallet.EncryptedPrivateKey,
		&wallet.EncryptionMethod,
		&wallet.KeyStorageLocation,
		&wallet.Label,
		&wallet.Description,
		&wallet.Metadata,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
		&wallet.CreatedBy,
		&wallet.UpdatedBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTreasuryWalletNotFound
		}
		return nil, fmt.Errorf("failed to get treasury wallet by address: %w", err)
	}

	return wallet, nil
}

// ListByType retrieves treasury wallets by type
func (r *TreasuryWalletRepository) ListByType(walletType model.WalletType) ([]*model.TreasuryWallet, error) {
	query := `
		SELECT
			id, wallet_type, blockchain, address, address_hash,
			is_multisig, multisig_scheme, multisig_signers, multisig_threshold,
			balance_crypto, balance_usd, balance_last_updated_at,
			sweep_threshold_usd, sweep_target_wallet_id, sweep_buffer_usd, auto_sweep_enabled,
			last_swept_at, last_deposit_at, last_withdrawal_at, total_swept_amount_usd,
			status, is_monitored,
			encrypted_private_key, encryption_method, key_storage_location,
			label, description, metadata,
			created_at, updated_at, created_by, updated_by
		FROM treasury_wallets
		WHERE wallet_type = $1
		ORDER BY blockchain, created_at
	`

	rows, err := r.db.Query(query, walletType)
	if err != nil {
		return nil, fmt.Errorf("failed to list treasury wallets by type: %w", err)
	}
	defer rows.Close()

	wallets := make([]*model.TreasuryWallet, 0)
	for rows.Next() {
		wallet := &model.TreasuryWallet{}
		err := rows.Scan(
			&wallet.ID,
			&wallet.WalletType,
			&wallet.Blockchain,
			&wallet.Address,
			&wallet.AddressHash,
			&wallet.IsMultisig,
			&wallet.MultisigScheme,
			pq.Array(&wallet.MultisigSigners),
			&wallet.MultisigThreshold,
			&wallet.BalanceCrypto,
			&wallet.BalanceUSD,
			&wallet.BalanceLastUpdatedAt,
			&wallet.SweepThresholdUSD,
			&wallet.SweepTargetWalletID,
			&wallet.SweepBufferUSD,
			&wallet.AutoSweepEnabled,
			&wallet.LastSweptAt,
			&wallet.LastDepositAt,
			&wallet.LastWithdrawalAt,
			&wallet.TotalSweptAmountUSD,
			&wallet.Status,
			&wallet.IsMonitored,
			&wallet.EncryptedPrivateKey,
			&wallet.EncryptionMethod,
			&wallet.KeyStorageLocation,
			&wallet.Label,
			&wallet.Description,
			&wallet.Metadata,
			&wallet.CreatedAt,
			&wallet.UpdatedAt,
			&wallet.CreatedBy,
			&wallet.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan treasury wallet: %w", err)
		}
		wallets = append(wallets, wallet)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating treasury wallets: %w", err)
	}

	return wallets, nil
}

// ListHotWalletsNeedingSweep retrieves hot wallets that need sweeping
func (r *TreasuryWalletRepository) ListHotWalletsNeedingSweep() ([]*model.TreasuryWallet, error) {
	query := `
		SELECT
			id, wallet_type, blockchain, address, address_hash,
			is_multisig, multisig_scheme, multisig_signers, multisig_threshold,
			balance_crypto, balance_usd, balance_last_updated_at,
			sweep_threshold_usd, sweep_target_wallet_id, sweep_buffer_usd, auto_sweep_enabled,
			last_swept_at, last_deposit_at, last_withdrawal_at, total_swept_amount_usd,
			status, is_monitored,
			encrypted_private_key, encryption_method, key_storage_location,
			label, description, metadata,
			created_at, updated_at, created_by, updated_by
		FROM treasury_wallets
		WHERE wallet_type = 'hot'
		  AND auto_sweep_enabled = TRUE
		  AND status = 'active'
		  AND balance_usd >= sweep_threshold_usd
		ORDER BY balance_usd DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list hot wallets needing sweep: %w", err)
	}
	defer rows.Close()

	wallets := make([]*model.TreasuryWallet, 0)
	for rows.Next() {
		wallet := &model.TreasuryWallet{}
		err := rows.Scan(
			&wallet.ID,
			&wallet.WalletType,
			&wallet.Blockchain,
			&wallet.Address,
			&wallet.AddressHash,
			&wallet.IsMultisig,
			&wallet.MultisigScheme,
			pq.Array(&wallet.MultisigSigners),
			&wallet.MultisigThreshold,
			&wallet.BalanceCrypto,
			&wallet.BalanceUSD,
			&wallet.BalanceLastUpdatedAt,
			&wallet.SweepThresholdUSD,
			&wallet.SweepTargetWalletID,
			&wallet.SweepBufferUSD,
			&wallet.AutoSweepEnabled,
			&wallet.LastSweptAt,
			&wallet.LastDepositAt,
			&wallet.LastWithdrawalAt,
			&wallet.TotalSweptAmountUSD,
			&wallet.Status,
			&wallet.IsMonitored,
			&wallet.EncryptedPrivateKey,
			&wallet.EncryptionMethod,
			&wallet.KeyStorageLocation,
			&wallet.Label,
			&wallet.Description,
			&wallet.Metadata,
			&wallet.CreatedAt,
			&wallet.UpdatedAt,
			&wallet.CreatedBy,
			&wallet.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan treasury wallet: %w", err)
		}
		wallets = append(wallets, wallet)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating treasury wallets: %w", err)
	}

	return wallets, nil
}

// Update updates an existing treasury wallet
func (r *TreasuryWalletRepository) Update(wallet *model.TreasuryWallet) error {
	if wallet == nil {
		return errors.New("treasury wallet cannot be nil")
	}
	if wallet.ID == uuid.Nil {
		return errors.New("invalid treasury wallet ID")
	}

	// Update address hash
	wallet.AddressHash = model.ComputeAddressHash(wallet.Address)

	query := `
		UPDATE treasury_wallets SET
			balance_crypto = $2,
			balance_usd = $3,
			balance_last_updated_at = $4,
			sweep_threshold_usd = $5,
			sweep_target_wallet_id = $6,
			sweep_buffer_usd = $7,
			auto_sweep_enabled = $8,
			last_swept_at = $9,
			last_deposit_at = $10,
			last_withdrawal_at = $11,
			total_swept_amount_usd = $12,
			status = $13,
			is_monitored = $14,
			label = $15,
			description = $16,
			metadata = $17,
			updated_at = $18,
			updated_by = $19
		WHERE id = $1
		RETURNING updated_at
	`

	wallet.UpdatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		wallet.ID,
		wallet.BalanceCrypto,
		wallet.BalanceUSD,
		wallet.BalanceLastUpdatedAt,
		wallet.SweepThresholdUSD,
		wallet.SweepTargetWalletID,
		wallet.SweepBufferUSD,
		wallet.AutoSweepEnabled,
		wallet.LastSweptAt,
		wallet.LastDepositAt,
		wallet.LastWithdrawalAt,
		wallet.TotalSweptAmountUSD,
		wallet.Status,
		wallet.IsMonitored,
		wallet.Label,
		wallet.Description,
		wallet.Metadata,
		wallet.UpdatedAt,
		wallet.UpdatedBy,
	).Scan(&wallet.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrTreasuryWalletNotFound
		}
		return fmt.Errorf("failed to update treasury wallet: %w", err)
	}

	return nil
}

// UpdateBalance updates the wallet balance
func (r *TreasuryWalletRepository) UpdateBalance(id uuid.UUID, balanceCrypto, balanceUSD decimal.Decimal) error {
	if id == uuid.Nil {
		return errors.New("invalid treasury wallet ID")
	}

	query := `
		UPDATE treasury_wallets
		SET
			balance_crypto = $2,
			balance_usd = $3,
			balance_last_updated_at = $4,
			updated_at = $4
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(query, id, balanceCrypto, balanceUSD, sql.NullTime{Time: now, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTreasuryWalletNotFound
	}

	return nil
}

// RecordSweep records a sweep operation on a hot wallet
func (r *TreasuryWalletRepository) RecordSweep(id uuid.UUID, sweptAmountUSD decimal.Decimal) error {
	if id == uuid.Nil {
		return errors.New("invalid treasury wallet ID")
	}

	query := `
		UPDATE treasury_wallets
		SET
			last_swept_at = $2,
			total_swept_amount_usd = total_swept_amount_usd + $3,
			updated_at = $2
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(query, id, sql.NullTime{Time: now, Valid: true}, sweptAmountUSD)
	if err != nil {
		return fmt.Errorf("failed to record sweep: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTreasuryWalletNotFound
	}

	return nil
}

// UpdateStatus updates the wallet status
func (r *TreasuryWalletRepository) UpdateStatus(id uuid.UUID, status model.WalletStatus) error {
	if id == uuid.Nil {
		return errors.New("invalid treasury wallet ID")
	}

	query := `
		UPDATE treasury_wallets
		SET
			status = $2,
			updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id, status, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update wallet status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTreasuryWalletNotFound
	}

	return nil
}

// Count returns the total number of treasury wallets
func (r *TreasuryWalletRepository) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM treasury_wallets`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count treasury wallets: %w", err)
	}

	return count, nil
}

// GetTotalBalance returns the total balance across all active wallets
func (r *TreasuryWalletRepository) GetTotalBalance() (map[string]decimal.Decimal, error) {
	query := `
		SELECT
			blockchain,
			SUM(balance_usd) as total_balance_usd
		FROM treasury_wallets
		WHERE status = 'active'
		GROUP BY blockchain
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get total balance: %w", err)
	}
	defer rows.Close()

	balances := make(map[string]decimal.Decimal)
	for rows.Next() {
		var blockchain string
		var balance decimal.Decimal
		if err := rows.Scan(&blockchain, &balance); err != nil {
			return nil, fmt.Errorf("failed to scan total balance: %w", err)
		}
		balances[blockchain] = balance
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating total balances: %w", err)
	}

	return balances, nil
}
