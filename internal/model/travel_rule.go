package model

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// TravelRuleData represents travel rule information for compliance with FATF recommendations
// Required for transactions > $1000 USD equivalent
type TravelRuleData struct {
	ID        string `json:"id" db:"id"`
	PaymentID string `json:"payment_id" db:"payment_id" validate:"required,uuid"`

	// Payer Information (Originator)
	PayerFullName      string         `json:"payer_full_name" db:"payer_full_name" validate:"required,min=2,max=255"`
	PayerWalletAddress string         `json:"payer_wallet_address" db:"payer_wallet_address" validate:"required,min=26,max=255"`
	PayerCountry       string         `json:"payer_country" db:"payer_country" validate:"required,iso3166_1_alpha2,len=2"`
	PayerIDDocument    sql.NullString `json:"payer_id_document,omitempty" db:"payer_id_document" validate:"omitempty,max=255"`

	// Merchant Information (Beneficiary)
	MerchantFullName string `json:"merchant_full_name" db:"merchant_full_name" validate:"required,min=2,max=255"`
	MerchantCountry  string `json:"merchant_country" db:"merchant_country" validate:"required,iso3166_1_alpha2,len=2"`

	// Transaction Information
	TransactionAmount   decimal.Decimal `json:"transaction_amount" db:"transaction_amount" validate:"required,gt=0"`
	TransactionCurrency string          `json:"transaction_currency" db:"transaction_currency" validate:"required,oneof=USD USDT USDC BUSD"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// IsValidForThreshold checks if the transaction amount meets the Travel Rule threshold ($1000 USD)
func (t *TravelRuleData) IsValidForThreshold() bool {
	threshold := decimal.NewFromInt(1000)
	return t.TransactionAmount.GreaterThanOrEqual(threshold)
}

// HasPayerIDDocument returns true if payer ID document is provided
func (t *TravelRuleData) HasPayerIDDocument() bool {
	return t.PayerIDDocument.Valid && t.PayerIDDocument.String != ""
}

// GetPayerIDDocument returns the payer ID document if available
func (t *TravelRuleData) GetPayerIDDocument() string {
	if t.PayerIDDocument.Valid {
		return t.PayerIDDocument.String
	}
	return ""
}

// IsSameCountry returns true if payer and merchant are from the same country
func (t *TravelRuleData) IsSameCountry() bool {
	return t.PayerCountry == t.MerchantCountry
}

// IsCrossBorder returns true if this is a cross-border transaction
func (t *TravelRuleData) IsCrossBorder() bool {
	return !t.IsSameCountry()
}
