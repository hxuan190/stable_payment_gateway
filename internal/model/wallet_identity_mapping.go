package model

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// WalletIdentityMapping represents the mapping between a wallet address and a user
// (PRD v2.2 - Smart Identity Mapping)
type WalletIdentityMapping struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`

	// Wallet Information
	WalletAddress     string `gorm:"type:varchar(255);not null;uniqueIndex:idx_wallet_blockchain" json:"wallet_address"`
	Blockchain        Chain  `gorm:"type:varchar(20);not null;uniqueIndex:idx_wallet_blockchain" json:"blockchain"`
	WalletAddressHash string `gorm:"type:varchar(64);not null;index" json:"wallet_address_hash"` // SHA-256 hash

	// User Link
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User   *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// KYC Status (denormalized for fast lookups)
	KYCStatus     KYCStatus    `gorm:"type:varchar(20);not null;default:'pending';index" json:"kyc_status"`
	KYCVerifiedAt sql.NullTime `gorm:"type:timestamp" json:"kyc_verified_at,omitempty"`

	// Activity Tracking
	FirstSeenAt   time.Time       `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"first_seen_at"`
	LastSeenAt    time.Time       `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;index" json:"last_seen_at"`
	PaymentCount  int             `gorm:"default:0" json:"payment_count"`
	TotalVolumeUSD decimal.Decimal `gorm:"type:decimal(20,2);default:0" json:"total_volume_usd"`
	LastPaymentID sql.NullString  `gorm:"type:uuid" json:"last_payment_id,omitempty"`

	// Risk Flags
	IsFlagged   bool           `gorm:"default:false;index" json:"is_flagged"`
	FlagReason  sql.NullString `gorm:"type:text" json:"flag_reason,omitempty"`
	FlaggedAt   sql.NullTime   `gorm:"type:timestamp" json:"flagged_at,omitempty"`

	// Metadata
	Metadata map[string]interface{} `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`

	// Audit fields
	CreatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;index" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name
func (WalletIdentityMapping) TableName() string {
	return "wallet_identity_mappings"
}

// BeforeCreate hook to generate wallet address hash
func (wm *WalletIdentityMapping) BeforeCreate() error {
	if wm.ID == uuid.Nil {
		wm.ID = uuid.New()
	}
	if wm.WalletAddressHash == "" {
		wm.WalletAddressHash = ComputeWalletHash(wm.WalletAddress)
	}
	return nil
}

// BeforeUpdate hook to update wallet address hash if address changed
func (wm *WalletIdentityMapping) BeforeUpdate() error {
	wm.WalletAddressHash = ComputeWalletHash(wm.WalletAddress)
	return nil
}

// ComputeWalletHash computes SHA-256 hash of wallet address for privacy
func ComputeWalletHash(address string) string {
	hash := sha256.Sum256([]byte(strings.ToLower(address)))
	return hex.EncodeToString(hash[:])
}

// IsKYCVerified returns true if the wallet's user has verified KYC
func (wm *WalletIdentityMapping) IsKYCVerified() bool {
	return wm.KYCStatus == KYCStatusApproved
}

// IsActive returns true if wallet was seen in last 7 days
func (wm *WalletIdentityMapping) IsActive() bool {
	return time.Since(wm.LastSeenAt) < 7*24*time.Hour
}

// UpdateActivity updates the activity tracking fields
func (wm *WalletIdentityMapping) UpdateActivity(paymentID uuid.UUID, amountUSD decimal.Decimal) {
	wm.LastSeenAt = time.Now()
	wm.PaymentCount++
	wm.TotalVolumeUSD = wm.TotalVolumeUSD.Add(amountUSD)
	wm.LastPaymentID = sql.NullString{String: paymentID.String(), Valid: true}
}

// Flag marks this wallet as flagged for review
func (wm *WalletIdentityMapping) Flag(reason string) {
	wm.IsFlagged = true
	wm.FlagReason = sql.NullString{String: reason, Valid: true}
	wm.FlaggedAt = sql.NullTime{Time: time.Now(), Valid: true}
}

// Unflag removes the flag from this wallet
func (wm *WalletIdentityMapping) Unflag() {
	wm.IsFlagged = false
	wm.FlagReason = sql.NullString{Valid: false}
	wm.FlaggedAt = sql.NullTime{Valid: false}
}

// GetCacheKey returns the Redis cache key for this wallet mapping
func (wm *WalletIdentityMapping) GetCacheKey() string {
	return "wallet:" + string(wm.Blockchain) + ":" + wm.WalletAddress
}

// GetCacheTTL returns the cache TTL in seconds (7 days)
func (wm *WalletIdentityMapping) GetCacheTTL() int {
	return 7 * 24 * 60 * 60 // 7 days
}
