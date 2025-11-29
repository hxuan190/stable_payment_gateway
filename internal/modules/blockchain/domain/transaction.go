package domain

import (
	"database/sql"
	"time"

	paymentDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/wallet/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/database"
	"github.com/shopspring/decimal"
)

// Network represents the blockchain network type
type Network = domain.Network

const (
	NetworkMainnet = domain.NetworkMainnet
	NetworkTestnet = domain.NetworkTestnet
	NetworkDevnet  = domain.NetworkDevnet
)

// BlockchainTxStatus represents the status of a blockchain transaction
type BlockchainTxStatus string

const (
	BlockchainTxStatusPending   BlockchainTxStatus = "pending"
	BlockchainTxStatusConfirmed BlockchainTxStatus = "confirmed"
	BlockchainTxStatusFinalized BlockchainTxStatus = "finalized"
	BlockchainTxStatusFailed    BlockchainTxStatus = "failed"
)

// BlockchainTransaction represents a blockchain transaction
type BlockchainTransaction struct {
	ID string `json:"id" db:"id"`

	// Blockchain details
	Chain   paymentDomain.Chain `json:"chain" db:"chain" validate:"required,oneof=solana bsc ethereum"`
	Network Network             `json:"network" db:"network" validate:"required,oneof=mainnet testnet devnet"`

	// Transaction identification
	TxHash         string        `json:"tx_hash" db:"tx_hash" validate:"required,min=32,max=255"`
	BlockNumber    sql.NullInt64 `json:"block_number,omitempty" db:"block_number"`
	BlockTimestamp sql.NullTime  `json:"block_timestamp,omitempty" db:"block_timestamp"`

	// Transaction details
	FromAddress string          `json:"from_address" db:"from_address" validate:"required,min=20,max=255"`
	ToAddress   string          `json:"to_address" db:"to_address" validate:"required,min=20,max=255"`
	Amount      decimal.Decimal `json:"amount" db:"amount" validate:"required,gt=0"`
	Currency    string          `json:"currency" db:"currency" validate:"required,min=2,max=10"`
	TokenMint   sql.NullString  `json:"token_mint,omitempty" db:"token_mint"`

	// Transaction memo/reference
	Memo                   sql.NullString `json:"memo,omitempty" db:"memo"`
	ParsedPaymentReference sql.NullString `json:"parsed_payment_reference,omitempty" db:"parsed_payment_reference"`

	// Confirmation status
	Confirmations int          `json:"confirmations" db:"confirmations" validate:"gte=0"`
	IsFinalized   bool         `json:"is_finalized" db:"is_finalized"`
	FinalizedAt   sql.NullTime `json:"finalized_at,omitempty" db:"finalized_at"`

	// Gas/fee information
	GasUsed     sql.NullInt64       `json:"gas_used,omitempty" db:"gas_used"`
	GasPrice    decimal.NullDecimal `json:"gas_price,omitempty" db:"gas_price"`
	TxFee       decimal.NullDecimal `json:"transaction_fee,omitempty" db:"transaction_fee"`
	FeeCurrency sql.NullString      `json:"fee_currency,omitempty" db:"fee_currency"`

	// Associated payment (if matched)
	PaymentID sql.NullString `json:"payment_id,omitempty" db:"payment_id"`
	IsMatched bool           `json:"is_matched" db:"is_matched"`
	MatchedAt sql.NullTime   `json:"matched_at,omitempty" db:"matched_at"`

	// Transaction status
	Status BlockchainTxStatus `json:"status" db:"status" validate:"required,oneof=pending confirmed finalized failed"`

	// Raw transaction data (for debugging)
	RawTransaction database.JSONBMap `json:"raw_transaction,omitempty" db:"raw_transaction"`

	// Error tracking
	ErrorMessage sql.NullString    `json:"error_message,omitempty" db:"error_message"`
	ErrorDetails database.JSONBMap `json:"error_details,omitempty" db:"error_details"`

	// Metadata
	Metadata database.JSONBMap `json:"metadata,omitempty" db:"metadata"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// IsPending returns true if the transaction is pending
func (b *BlockchainTransaction) IsPending() bool {
	return b.Status == BlockchainTxStatusPending
}

// IsConfirmed returns true if the transaction is confirmed or finalized
func (b *BlockchainTransaction) IsConfirmed() bool {
	return b.Status == BlockchainTxStatusConfirmed || b.Status == BlockchainTxStatusFinalized
}

// IsFailed returns true if the transaction failed
func (b *BlockchainTransaction) IsFailed() bool {
	return b.Status == BlockchainTxStatusFailed
}

// HasPaymentReference returns true if the transaction has a payment reference
func (b *BlockchainTransaction) HasPaymentReference() bool {
	return b.ParsedPaymentReference.Valid && b.ParsedPaymentReference.String != ""
}

// GetPaymentReference returns the payment reference if available
func (b *BlockchainTransaction) GetPaymentReference() string {
	if b.ParsedPaymentReference.Valid {
		return b.ParsedPaymentReference.String
	}
	return ""
}

// GetPaymentID returns the payment ID if available
func (b *BlockchainTransaction) GetPaymentID() string {
	if b.PaymentID.Valid {
		return b.PaymentID.String
	}
	return ""
}

// GetMemo returns the transaction memo if available
func (b *BlockchainTransaction) GetMemo() string {
	if b.Memo.Valid {
		return b.Memo.String
	}
	return ""
}

// GetErrorMessage returns the error message if available
func (b *BlockchainTransaction) GetErrorMessage() string {
	if b.ErrorMessage.Valid {
		return b.ErrorMessage.String
	}
	return ""
}

// GetTokenMint returns the token mint address if available
func (b *BlockchainTransaction) GetTokenMint() string {
	if b.TokenMint.Valid {
		return b.TokenMint.String
	}
	return ""
}

// CanBeMatched returns true if the transaction can be matched to a payment
func (b *BlockchainTransaction) CanBeMatched() bool {
	return !b.IsMatched && b.HasPaymentReference() && b.IsConfirmed()
}
