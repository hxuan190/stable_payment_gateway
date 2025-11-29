package domain

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
	paymentDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/database"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// WalletType represents the type of treasury wallet
type WalletType string

const (
	WalletTypeHot  WalletType = "hot"  // Online wallet for active use
	WalletTypeCold WalletType = "cold" // Offline wallet for long-term storage
)

// WalletStatus represents the wallet's operational status
type WalletStatus string

const (
	WalletStatusActive     WalletStatus = "active"
	WalletStatusSuspended  WalletStatus = "suspended"
	WalletStatusDeprecated WalletStatus = "deprecated"
)

// TreasuryWallet represents a hot or cold cryptocurrency wallet (PRD v2.2 - Treasury)
type TreasuryWallet struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`

	// Wallet Type
	WalletType WalletType `gorm:"type:varchar(20);not null;index" json:"wallet_type"`

	// Blockchain Information
	Blockchain  paymentDomain.Chain `gorm:"type:varchar(20);not null;uniqueIndex:idx_blockchain_address" json:"blockchain"`
	Address     string              `gorm:"type:varchar(255);not null;uniqueIndex:idx_blockchain_address" json:"address"`
	AddressHash string              `gorm:"type:varchar(64);not null;index" json:"address_hash"` // SHA-256

	// Multi-Sig Configuration (for cold wallets)
	IsMultisig        bool           `gorm:"default:false" json:"is_multisig"`
	MultisigScheme    sql.NullString `gorm:"type:varchar(20)" json:"multisig_scheme,omitempty"` // '2-of-3', '3-of-5'
	MultisigSigners   pq.StringArray `gorm:"type:text[]" json:"multisig_signers,omitempty"`     // Array of signer addresses
	MultisigThreshold sql.NullInt32  `gorm:"type:integer" json:"multisig_threshold,omitempty"`  // Number of signatures required

	// Balance Tracking
	BalanceCrypto        decimal.Decimal `gorm:"type:decimal(30,18);default:0" json:"balance_crypto"` // Native token/coin
	BalanceUSD           decimal.Decimal `gorm:"type:decimal(20,2);default:0" json:"balance_usd"`
	BalanceLastUpdatedAt sql.NullTime    `gorm:"type:timestamp;index" json:"balance_last_updated_at,omitempty"`

	// Sweeping Configuration (for hot wallets)
	SweepThresholdUSD   decimal.Decimal `gorm:"type:decimal(20,2);default:10000" json:"sweep_threshold_usd"` // Sweep when > this
	SweepTargetWalletID sql.NullString  `gorm:"type:uuid" json:"sweep_target_wallet_id,omitempty"`           // Target cold wallet
	SweepBufferUSD      decimal.Decimal `gorm:"type:decimal(20,2);default:5000" json:"sweep_buffer_usd"`     // Leave this much
	AutoSweepEnabled    bool            `gorm:"default:true" json:"auto_sweep_enabled"`

	// Activity Tracking
	LastSweptAt         sql.NullTime    `gorm:"type:timestamp" json:"last_swept_at,omitempty"`
	LastDepositAt       sql.NullTime    `gorm:"type:timestamp" json:"last_deposit_at,omitempty"`
	LastWithdrawalAt    sql.NullTime    `gorm:"type:timestamp" json:"last_withdrawal_at,omitempty"`
	TotalSweptAmountUSD decimal.Decimal `gorm:"type:decimal(20,2);default:0" json:"total_swept_amount_usd"`

	// Status
	Status      WalletStatus `gorm:"type:varchar(20);default:'active';index" json:"status"`
	IsMonitored bool         `gorm:"default:true" json:"is_monitored"`

	// Security
	EncryptedPrivateKey sql.NullString `gorm:"type:text" json:"encrypted_private_key,omitempty"`    // For hot wallets only
	EncryptionMethod    sql.NullString `gorm:"type:varchar(50)" json:"encryption_method,omitempty"` // 'AWS-KMS', 'Vault'
	KeyStorageLocation  sql.NullString `gorm:"type:varchar(100)" json:"key_storage_location,omitempty"`

	// Metadata
	Label       string            `gorm:"type:varchar(255);index" json:"label"` // e.g., "TRON Hot Wallet #1"
	Description sql.NullString    `gorm:"type:text" json:"description,omitempty"`
	Metadata    database.JSONBMap `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`

	// Audit fields
	CreatedAt time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
	CreatedBy sql.NullString `gorm:"type:varchar(100)" json:"created_by,omitempty"`
	UpdatedBy sql.NullString `gorm:"type:varchar(100)" json:"updated_by,omitempty"`
}

// TableName specifies the table name
func (TreasuryWallet) TableName() string {
	return "treasury_wallets"
}

// BeforeCreate hook
func (tw *TreasuryWallet) BeforeCreate() error {
	if tw.ID == uuid.Nil {
		tw.ID = uuid.New()
	}
	if tw.AddressHash == "" {
		tw.AddressHash = ComputeAddressHash(tw.Address)
	}
	return nil
}

// BeforeUpdate hook
func (tw *TreasuryWallet) BeforeUpdate() error {
	tw.AddressHash = ComputeAddressHash(tw.Address)
	return nil
}

// ComputeAddressHash computes SHA-256 hash of address
func ComputeAddressHash(address string) string {
	hash := sha256.Sum256([]byte(strings.ToLower(address)))
	return hex.EncodeToString(hash[:])
}

// IsActive returns true if wallet is active
func (tw *TreasuryWallet) IsActive() bool {
	return tw.Status == WalletStatusActive
}

// IsHotWallet returns true if this is a hot wallet
func (tw *TreasuryWallet) IsHotWallet() bool {
	return tw.WalletType == WalletTypeHot
}

// IsColdWallet returns true if this is a cold wallet
func (tw *TreasuryWallet) IsColdWallet() bool {
	return tw.WalletType == WalletTypeCold
}

// NeedsSweep returns true if hot wallet balance exceeds threshold
func (tw *TreasuryWallet) NeedsSweep() bool {
	return tw.IsHotWallet() && tw.AutoSweepEnabled && tw.IsActive() &&
		tw.BalanceUSD.GreaterThanOrEqual(tw.SweepThresholdUSD)
}

// CalculateSweepAmount calculates how much to sweep (balance - buffer)
func (tw *TreasuryWallet) CalculateSweepAmount() decimal.Decimal {
	if !tw.NeedsSweep() {
		return decimal.Zero
	}
	return tw.BalanceUSD.Sub(tw.SweepBufferUSD)
}

// UpdateBalance updates the wallet balance
func (tw *TreasuryWallet) UpdateBalance(balanceCrypto, balanceUSD decimal.Decimal) {
	tw.BalanceCrypto = balanceCrypto
	tw.BalanceUSD = balanceUSD
	tw.BalanceLastUpdatedAt = sql.NullTime{Time: time.Now(), Valid: true}
}

// TreasuryOperationType represents the type of treasury operation
type TreasuryOperationType string

const (
	TreasuryOpSweep               TreasuryOperationType = "sweep"
	TreasuryOpManualTransfer      TreasuryOperationType = "manual_transfer"
	TreasuryOpOTCSettlement       TreasuryOperationType = "otc_settlement"
	TreasuryOpConsolidation       TreasuryOperationType = "consolidation"
	TreasuryOpEmergencyWithdrawal TreasuryOperationType = "emergency_withdrawal"
)

// TreasuryOperationStatus represents the operation status
type TreasuryOperationStatus string

const (
	TreasuryOpStatusInitiated         TreasuryOperationStatus = "initiated"
	TreasuryOpStatusPendingSignatures TreasuryOperationStatus = "pending_signatures"
	TreasuryOpStatusBroadcasted       TreasuryOperationStatus = "broadcasted"
	TreasuryOpStatusConfirmed         TreasuryOperationStatus = "confirmed"
	TreasuryOpStatusFailed            TreasuryOperationStatus = "failed"
	TreasuryOpStatusCancelled         TreasuryOperationStatus = "cancelled"
)

// TreasuryOperation logs all treasury operations (PRD v2.2 - Treasury)
type TreasuryOperation struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`

	// Operation Type
	OperationType TreasuryOperationType `gorm:"type:varchar(50);not null;index" json:"operation_type"`

	// Wallet Information
	FromWalletID sql.NullString `gorm:"type:uuid;index" json:"from_wallet_id,omitempty"`
	ToWalletID   sql.NullString `gorm:"type:uuid;index" json:"to_wallet_id,omitempty"`
	FromAddress  sql.NullString `gorm:"type:varchar(255)" json:"from_address,omitempty"`
	ToAddress    sql.NullString `gorm:"type:varchar(255)" json:"to_address,omitempty"`

	// Blockchain & Transaction
	Blockchain      paymentDomain.Chain `gorm:"type:varchar(20);not null;index" json:"blockchain"`
	Amount          decimal.Decimal     `gorm:"type:decimal(30,18);not null" json:"amount"`
	Currency        string              `gorm:"type:varchar(10);not null" json:"currency"` // 'TRX', 'SOL', 'BNB', 'USDT', 'USDC'
	AmountUSD       decimal.Decimal     `gorm:"type:decimal(20,2)" json:"amount_usd,omitempty"`
	TxHash          sql.NullString      `gorm:"type:varchar(255);index" json:"tx_hash,omitempty"`
	TxConfirmations int                 `gorm:"default:0" json:"tx_confirmations"`

	// Multi-Sig
	RequiresMultisig    bool              `gorm:"default:false" json:"requires_multisig"`
	SignaturesRequired  sql.NullInt32     `gorm:"type:integer" json:"signatures_required,omitempty"`
	SignaturesCollected int               `gorm:"default:0" json:"signatures_collected"`
	Signers             database.JSONBMap `gorm:"type:jsonb" json:"signers,omitempty"` // {signer_address, signed_at, signature}

	// Status
	Status        TreasuryOperationStatus `gorm:"type:varchar(20);not null;default:'initiated';index" json:"status"`
	InitiatedAt   time.Time               `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;index" json:"initiated_at"`
	BroadcastedAt sql.NullTime            `gorm:"type:timestamp" json:"broadcasted_at,omitempty"`
	ConfirmedAt   sql.NullTime            `gorm:"type:timestamp;index" json:"confirmed_at,omitempty"`
	FailedAt      sql.NullTime            `gorm:"type:timestamp" json:"failed_at,omitempty"`
	CancelledAt   sql.NullTime            `gorm:"type:timestamp" json:"cancelled_at,omitempty"`

	// Reason & Context
	Reason      sql.NullString `gorm:"type:text" json:"reason,omitempty"`
	TriggeredBy sql.NullString `gorm:"type:varchar(50)" json:"triggered_by,omitempty"` // 'auto_sweep', 'manual', 'threshold'
	InitiatedBy sql.NullString `gorm:"type:varchar(100)" json:"initiated_by,omitempty"`

	// Error Handling
	ErrorMessage sql.NullString `gorm:"type:text" json:"error_message,omitempty"`
	ErrorCode    sql.NullString `gorm:"type:varchar(50)" json:"error_code,omitempty"`
	RetryCount   int            `gorm:"default:0" json:"retry_count"`

	// Gas/Fee
	GasUsed           decimal.Decimal `gorm:"type:decimal(30,18)" json:"gas_used,omitempty"`
	GasPrice          decimal.Decimal `gorm:"type:decimal(30,18)" json:"gas_price,omitempty"`
	TransactionFee    decimal.Decimal `gorm:"type:decimal(30,18)" json:"transaction_fee,omitempty"`
	TransactionFeeUSD decimal.Decimal `gorm:"type:decimal(20,2)" json:"transaction_fee_usd,omitempty"`

	// Metadata
	Metadata database.JSONBMap `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`

	// Audit fields
	CreatedAt time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
	CreatedBy sql.NullString `gorm:"type:varchar(100)" json:"created_by,omitempty"`
	UpdatedBy sql.NullString `gorm:"type:varchar(100)" json:"updated_by,omitempty"`
}

// TableName specifies the table name
func (TreasuryOperation) TableName() string {
	return "treasury_operations"
}

// BeforeCreate hook
func (to *TreasuryOperation) BeforeCreate() error {
	if to.ID == uuid.Nil {
		to.ID = uuid.New()
	}
	return nil
}

// IsConfirmed returns true if operation is confirmed
func (to *TreasuryOperation) IsConfirmed() bool {
	return to.Status == TreasuryOpStatusConfirmed
}

// IsPending returns true if operation is pending
func (to *TreasuryOperation) IsPending() bool {
	return to.Status == TreasuryOpStatusInitiated || to.Status == TreasuryOpStatusPendingSignatures || to.Status == TreasuryOpStatusBroadcasted
}

// RequiresSignatures returns true if operation needs more signatures
func (to *TreasuryOperation) RequiresSignatures() bool {
	return to.RequiresMultisig && to.Status == TreasuryOpStatusPendingSignatures &&
		(to.SignaturesRequired.Valid && int(to.SignaturesRequired.Int32) > to.SignaturesCollected)
}

// MarkAsBroadcasted updates status to broadcasted
func (to *TreasuryOperation) MarkAsBroadcasted(txHash string) {
	to.Status = TreasuryOpStatusBroadcasted
	to.BroadcastedAt = sql.NullTime{Time: time.Now(), Valid: true}
	if txHash != "" {
		to.TxHash = sql.NullString{String: txHash, Valid: true}
	}
}

// MarkAsConfirmed updates status to confirmed
func (to *TreasuryOperation) MarkAsConfirmed() {
	to.Status = TreasuryOpStatusConfirmed
	to.ConfirmedAt = sql.NullTime{Time: time.Now(), Valid: true}
}

// MarkAsFailed updates status to failed
func (to *TreasuryOperation) MarkAsFailed(errorMessage, errorCode string) {
	to.Status = TreasuryOpStatusFailed
	to.FailedAt = sql.NullTime{Time: time.Now(), Valid: true}
	to.ErrorMessage = sql.NullString{String: errorMessage, Valid: true}
	if errorCode != "" {
		to.ErrorCode = sql.NullString{String: errorCode, Valid: true}
	}
}

// AddSignature adds a signature to the multisig operation
func (to *TreasuryOperation) AddSignature(signerAddress, signature string) {
	to.SignaturesCollected++
	if to.Signers == nil {
		to.Signers = make(database.JSONBMap)
	}
	to.Signers[signerAddress] = map[string]interface{}{
		"signature": signature,
		"signed_at": time.Now(),
	}

	// Check if we have enough signatures
	if to.SignaturesRequired.Valid && to.SignaturesCollected >= int(to.SignaturesRequired.Int32) {
		to.Status = TreasuryOpStatusInitiated // Ready to broadcast
	}
}
