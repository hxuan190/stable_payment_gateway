package model

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// TravelRuleData represents travel rule information for compliance with FATF recommendations
// Required for transactions > $1000 USD equivalent
// Reference: FATF Recommendation 16 (Wire Transfers)
type TravelRuleData struct {
	ID        string `json:"id" db:"id"`
	PaymentID string `json:"payment_id" db:"payment_id" validate:"required,uuid"`

	// Payer Information (Originator) - FATF Required
	// Sensitive fields are encrypted at rest using AES-256-GCM
	PayerFullName      string         `json:"payer_full_name" db:"payer_full_name" gorm:"type:text" encrypt:"true" validate:"required,min=2,max=255"`
	PayerWalletAddress string         `json:"payer_wallet_address" db:"payer_wallet_address" gorm:"type:text" validate:"required,min=26,max=255"`
	PayerCountry       string         `json:"payer_country" db:"payer_country" gorm:"type:varchar(2)" validate:"required,iso3166_1_alpha2,len=2"`
	PayerIDDocument    sql.NullString `json:"payer_id_document,omitempty" db:"payer_id_document" gorm:"type:text" encrypt:"true" validate:"omitempty,max=255"`
	PayerDateOfBirth   sql.NullTime   `json:"payer_date_of_birth,omitempty" db:"payer_date_of_birth" encrypt:"true"`
	PayerAddress       sql.NullString `json:"payer_address,omitempty" db:"payer_address" gorm:"type:text" encrypt:"true"`

	// Merchant Information (Beneficiary) - FATF Required
	MerchantFullName             string         `json:"merchant_full_name" db:"merchant_full_name" validate:"required,min=2,max=255"`
	MerchantCountry              string         `json:"merchant_country" db:"merchant_country" validate:"required,iso3166_1_alpha2,len=2"`
	MerchantWalletAddress        sql.NullString `json:"merchant_wallet_address,omitempty" db:"merchant_wallet_address"`
	MerchantIDDocument           sql.NullString `json:"merchant_id_document,omitempty" db:"merchant_id_document"`
	MerchantBusinessRegistration sql.NullString `json:"merchant_business_registration,omitempty" db:"merchant_business_registration"`
	MerchantAddress              sql.NullString `json:"merchant_address,omitempty" db:"merchant_address"`

	// Transaction Information
	TransactionAmount   decimal.Decimal `json:"transaction_amount" db:"transaction_amount" validate:"required,gt=0"`
	TransactionCurrency string          `json:"transaction_currency" db:"transaction_currency" validate:"required,oneof=USD USDT USDC BUSD"`
	TransactionPurpose  sql.NullString  `json:"transaction_purpose,omitempty" db:"transaction_purpose" validate:"omitempty,oneof=payment_for_goods payment_for_services hotel_accommodation restaurant_payment tourism_service transport_service entertainment other"`

	// Risk Assessment
	RiskLevel             sql.NullString    `json:"risk_level,omitempty" db:"risk_level" validate:"omitempty,oneof=low medium high critical"`
	RiskScore             decimal.NullDecimal `json:"risk_score,omitempty" db:"risk_score"`
	ScreeningStatus       sql.NullString    `json:"screening_status,omitempty" db:"screening_status" validate:"omitempty,oneof=pending in_progress completed flagged cleared"`
	ScreeningCompletedAt  sql.NullTime      `json:"screening_completed_at,omitempty" db:"screening_completed_at"`

	// Regulatory Reporting
	ReportedToAuthority bool           `json:"reported_to_authority" db:"reported_to_authority"`
	ReportedAt          sql.NullTime   `json:"reported_at,omitempty" db:"reported_at"`
	ReportReference     sql.NullString `json:"report_reference,omitempty" db:"report_reference"`

	// Data Retention
	RetentionPolicy string       `json:"retention_policy" db:"retention_policy" validate:"omitempty,oneof=standard extended permanent"`
	ArchivedAt      sql.NullTime `json:"archived_at,omitempty" db:"archived_at"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
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

// GetTransactionPurpose returns the transaction purpose if available
func (t *TravelRuleData) GetTransactionPurpose() string {
	if t.TransactionPurpose.Valid {
		return t.TransactionPurpose.String
	}
	return "other"
}

// GetRiskLevel returns the risk level, defaulting to "low" if not set
func (t *TravelRuleData) GetRiskLevel() string {
	if t.RiskLevel.Valid {
		return t.RiskLevel.String
	}
	return "low"
}

// IsHighRisk returns true if the transaction is high or critical risk
func (t *TravelRuleData) IsHighRisk() bool {
	riskLevel := t.GetRiskLevel()
	return riskLevel == "high" || riskLevel == "critical"
}

// RequiresManualReview returns true if the transaction requires manual compliance review
func (t *TravelRuleData) RequiresManualReview() bool {
	// Manual review required for:
	// 1. High risk transactions
	// 2. Cross-border transactions with high amounts (>$10,000)
	// 3. Flagged during screening
	if t.IsHighRisk() {
		return true
	}

	if t.IsCrossBorder() && t.TransactionAmount.GreaterThan(decimal.NewFromInt(10000)) {
		return true
	}

	if t.ScreeningStatus.Valid && t.ScreeningStatus.String == "flagged" {
		return true
	}

	return false
}

// IsScreeningComplete returns true if AML screening is completed
func (t *TravelRuleData) IsScreeningComplete() bool {
	if !t.ScreeningStatus.Valid {
		return false
	}
	status := t.ScreeningStatus.String
	return status == "completed" || status == "cleared"
}

// HasMerchantDetails returns true if merchant wallet and business details are provided
func (t *TravelRuleData) HasMerchantDetails() bool {
	return t.MerchantWalletAddress.Valid && t.MerchantBusinessRegistration.Valid
}

// GetMerchantWalletAddress returns the merchant wallet address if available
func (t *TravelRuleData) GetMerchantWalletAddress() string {
	if t.MerchantWalletAddress.Valid {
		return t.MerchantWalletAddress.String
	}
	return ""
}

// IsReportedToAuthority returns true if this transaction was reported to regulatory authority
func (t *TravelRuleData) IsReportedToAuthority() bool {
	return t.ReportedToAuthority && t.ReportedAt.Valid
}

// GetRetentionPeriodYears returns the retention period in years based on policy
func (t *TravelRuleData) GetRetentionPeriodYears() int {
	switch t.RetentionPolicy {
	case "standard":
		return 7 // FATF minimum recommendation
	case "extended":
		return 10
	case "permanent":
		return 999 // Effectively permanent
	default:
		return 7 // Default to standard
	}
}

// IsArchived returns true if the data has been archived to cold storage
func (t *TravelRuleData) IsArchived() bool {
	return t.ArchivedAt.Valid
}

// ShouldBeArchived returns true if the data should be archived based on age
func (t *TravelRuleData) ShouldBeArchived() bool {
	// Archive after 12 months for standard retention
	if t.RetentionPolicy == "permanent" {
		return false
	}

	archiveThreshold := 12 * 30 * 24 * time.Hour // ~12 months
	return time.Since(t.CreatedAt) > archiveThreshold
}

// TravelRuleSummary provides a summary of travel rule data for display
type TravelRuleSummary struct {
	PaymentID         string          `json:"payment_id"`
	OriginatorName    string          `json:"originator_name"`
	OriginatorCountry string          `json:"originator_country"`
	BeneficiaryName   string          `json:"beneficiary_name"`
	BeneficiaryCountry string         `json:"beneficiary_country"`
	Amount            decimal.Decimal `json:"amount"`
	Currency          string          `json:"currency"`
	TransactionDate   time.Time       `json:"transaction_date"`
	IsCrossBorder     bool            `json:"is_cross_border"`
	RiskLevel         string          `json:"risk_level"`
	IsReported        bool            `json:"is_reported"`
}

// ToSummary converts TravelRuleData to a summary format
func (t *TravelRuleData) ToSummary() *TravelRuleSummary {
	return &TravelRuleSummary{
		PaymentID:         t.PaymentID,
		OriginatorName:    t.PayerFullName,
		OriginatorCountry: t.PayerCountry,
		BeneficiaryName:   t.MerchantFullName,
		BeneficiaryCountry: t.MerchantCountry,
		Amount:            t.TransactionAmount,
		Currency:          t.TransactionCurrency,
		TransactionDate:   t.CreatedAt,
		IsCrossBorder:     t.IsCrossBorder(),
		RiskLevel:         t.GetRiskLevel(),
		IsReported:        t.IsReportedToAuthority(),
	}
}
