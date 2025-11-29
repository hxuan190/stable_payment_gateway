package repository

import (
	"errors"
	"fmt"
	"time"

	blockchainDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/blockchain/domain"
	paymentDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
	"gorm.io/gorm"
)

var (
	// ErrBlockchainTxNotFound is returned when a blockchain transaction is not found
	ErrBlockchainTxNotFound = errors.New("blockchain transaction not found")
	// ErrBlockchainTxAlreadyExists is returned when a blockchain transaction with the same tx hash already exists
	ErrBlockchainTxAlreadyExists = errors.New("blockchain transaction with this tx hash already exists")
	// ErrInvalidTxHash is returned when the tx hash is invalid
	ErrInvalidTxHash = errors.New("invalid transaction hash")
	// ErrInvalidBlockchainTxStatus is returned when the blockchain transaction status is invalid
	ErrInvalidBlockchainTxStatus = errors.New("invalid blockchain transaction status")
)

// BlockchainTxRepository handles database operations for blockchain transactions
type BlockchainTxRepository struct {
	db *gorm.DB
}

// NewBlockchainTxRepository creates a new blockchain transaction repository
func NewBlockchainTxRepository(db *gorm.DB) *BlockchainTxRepository {
	return &BlockchainTxRepository{
		db: db,
	}
}

// Create inserts a new blockchain transaction into the database
func (r *BlockchainTxRepository) Create(tx *blockchainDomain.BlockchainTransaction) error {
	if tx == nil {
		return errors.New("blockchain transaction cannot be nil")
	}

	if tx.TxHash == "" {
		return ErrInvalidTxHash
	}

	if err := r.db.Create(tx).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrBlockchainTxAlreadyExists
		}
		return fmt.Errorf("failed to create blockchain transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a blockchain transaction by its ID
func (r *BlockchainTxRepository) GetByID(id string) (*blockchainDomain.BlockchainTransaction, error) {
	if id == "" {
		return nil, errors.New("id cannot be empty")
	}

	tx := &blockchainDomain.BlockchainTransaction{}
	if err := r.db.Where("id = ?", id).First(tx).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBlockchainTxNotFound
		}
		return nil, fmt.Errorf("failed to get blockchain transaction by ID: %w", err)
	}

	return tx, nil
}

// GetByTxHash retrieves a blockchain transaction by its transaction hash
func (r *BlockchainTxRepository) GetByTxHash(txHash string) (*blockchainDomain.BlockchainTransaction, error) {
	if txHash == "" {
		return nil, ErrInvalidTxHash
	}

	tx := &blockchainDomain.BlockchainTransaction{}
	if err := r.db.Where("tx_hash = ?", txHash).First(tx).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBlockchainTxNotFound
		}
		return nil, fmt.Errorf("failed to get blockchain transaction by tx hash: %w", err)
	}

	return tx, nil
}

// UpdateConfirmations updates the confirmation count for a blockchain transaction
func (r *BlockchainTxRepository) UpdateConfirmations(txHash string, confirmations int) error {
	if txHash == "" {
		return ErrInvalidTxHash
	}

	if confirmations < 0 {
		return errors.New("confirmations cannot be negative")
	}

	result := r.db.Model(&blockchainDomain.BlockchainTransaction{}).
		Where("tx_hash = ?", txHash).
		Updates(map[string]interface{}{
			"confirmations": confirmations,
			"updated_at":    time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update confirmations: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrBlockchainTxNotFound
	}

	return nil
}

// UpdateStatus updates the status of a blockchain transaction
func (r *BlockchainTxRepository) UpdateStatus(txHash string, status blockchainDomain.BlockchainTxStatus) error {
	if txHash == "" {
		return ErrInvalidTxHash
	}

	result := r.db.Model(&blockchainDomain.BlockchainTransaction{}).
		Where("tx_hash = ?", txHash).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrBlockchainTxNotFound
	}

	return nil
}

// MarkAsFinalized marks a blockchain transaction as finalized
func (r *BlockchainTxRepository) MarkAsFinalized(txHash string) error {
	if txHash == "" {
		return ErrInvalidTxHash
	}

	now := time.Now()
	result := r.db.Model(&blockchainDomain.BlockchainTransaction{}).
		Where("tx_hash = ?", txHash).
		Updates(map[string]interface{}{
			"is_finalized": true,
			"finalized_at": now,
			"status":       blockchainDomain.BlockchainTxStatusFinalized,
			"updated_at":   now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark as finalized: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrBlockchainTxNotFound
	}

	return nil
}

// MatchToPayment associates a blockchain transaction with a payment
func (r *BlockchainTxRepository) MatchToPayment(txHash string, paymentID string) error {
	if txHash == "" {
		return ErrInvalidTxHash
	}

	if paymentID == "" {
		return errors.New("payment ID cannot be empty")
	}

	now := time.Now()
	result := r.db.Model(&blockchainDomain.BlockchainTransaction{}).
		Where("tx_hash = ?", txHash).
		Updates(map[string]interface{}{
			"payment_id": paymentID,
			"is_matched": true,
			"matched_at": now,
			"updated_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to match to payment: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrBlockchainTxNotFound
	}

	return nil
}

// MarkAsFailed marks a blockchain transaction as failed with an error message
func (r *BlockchainTxRepository) MarkAsFailed(txHash string, errorMessage string) error {
	if txHash == "" {
		return ErrInvalidTxHash
	}

	result := r.db.Model(&blockchainDomain.BlockchainTransaction{}).
		Where("tx_hash = ?", txHash).
		Updates(map[string]interface{}{
			"status":        blockchainDomain.BlockchainTxStatusFailed,
			"error_message": errorMessage,
			"updated_at":    time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark as failed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrBlockchainTxNotFound
	}

	return nil
}

// GetPendingTransactions retrieves all pending blockchain transactions
func (r *BlockchainTxRepository) GetPendingTransactions() ([]*blockchainDomain.BlockchainTransaction, error) {
	var transactions []*blockchainDomain.BlockchainTransaction
	if err := r.db.Where("status = ? AND is_finalized = ?", blockchainDomain.BlockchainTxStatusPending, false).
		Order("created_at ASC").
		Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending transactions: %w", err)
	}

	return transactions, nil
}

// GetUnmatchedTransactions retrieves blockchain transactions that have a payment reference but haven't been matched yet
func (r *BlockchainTxRepository) GetUnmatchedTransactions(chain paymentDomain.Chain) ([]*blockchainDomain.BlockchainTransaction, error) {
	var transactions []*blockchainDomain.BlockchainTransaction
	if err := r.db.Where("chain = ? AND is_matched = ? AND parsed_payment_reference IS NOT NULL AND status IN ?",
		chain, false, []blockchainDomain.BlockchainTxStatus{blockchainDomain.BlockchainTxStatusConfirmed, blockchainDomain.BlockchainTxStatusFinalized}).
		Order("created_at ASC").
		Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get unmatched transactions: %w", err)
	}

	return transactions, nil
}

// GetByPaymentID retrieves all blockchain transactions associated with a payment
func (r *BlockchainTxRepository) GetByPaymentID(paymentID string) ([]*blockchainDomain.BlockchainTransaction, error) {
	if paymentID == "" {
		return nil, errors.New("payment ID cannot be empty")
	}

	var transactions []*blockchainDomain.BlockchainTransaction
	if err := r.db.Where("payment_id = ?", paymentID).
		Order("created_at DESC").
		Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get transactions by payment ID: %w", err)
	}

	return transactions, nil
}

// ListByChainAndStatus retrieves blockchain transactions filtered by chain and status
func (r *BlockchainTxRepository) ListByChainAndStatus(chain paymentDomain.Chain, status blockchainDomain.BlockchainTxStatus, limit, offset int) ([]*blockchainDomain.BlockchainTransaction, error) {
	var transactions []*blockchainDomain.BlockchainTransaction
	if err := r.db.Where("chain = ? AND status = ?", chain, status).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to list transactions by chain and status: %w", err)
	}

	return transactions, nil
}
