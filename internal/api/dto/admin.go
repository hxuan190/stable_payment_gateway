package dto

import (
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/shopspring/decimal"
)

// Admin Merchant Management DTOs

// ApproveKYCRequest represents a request to approve merchant KYC
type ApproveKYCRequest struct {
	ApprovedBy string `json:"approved_by" binding:"required" example:"admin@paymentgateway.com"`
}

// RejectKYCRequest represents a request to reject merchant KYC
type RejectKYCRequest struct {
	Reason string `json:"reason" binding:"required" example:"Invalid business license document"`
}

// ListMerchantsQuery represents query parameters for listing merchants
type ListMerchantsQuery struct {
	KYCStatus string `form:"kyc_status" binding:"omitempty,oneof=pending approved rejected" example:"pending"`
	Limit     int    `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
	Offset    int    `form:"offset" binding:"omitempty,min=0" example:"0"`
}

// MerchantDetailResponse represents detailed merchant information for admin
type MerchantDetailResponse struct {
	ID                    string       `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email                 string       `json:"email" example:"merchant@example.com"`
	BusinessName          string       `json:"business_name" example:"Da Nang Beach Resort"`
	BusinessTaxID         string       `json:"business_tax_id" example:"0123456789"`
	BusinessLicenseNumber string       `json:"business_license_number" example:"DN-2024-001"`
	OwnerFullName         string       `json:"owner_full_name" example:"Nguyen Van A"`
	OwnerIDCard           string       `json:"owner_id_card" example:"001234567890"`
	PhoneNumber           string       `json:"phone_number" example:"+84901234567"`
	BusinessAddress       string       `json:"business_address" example:"123 Bach Dang, Da Nang"`
	City                  string       `json:"city" example:"Da Nang"`
	District              string       `json:"district" example:"Son Tra"`
	BankAccountName       string       `json:"bank_account_name" example:"NGUYEN VAN A"`
	BankAccountNumber     string       `json:"bank_account_number" example:"1234567890"`
	BankName              string       `json:"bank_name" example:"Vietcombank"`
	BankBranch            string       `json:"bank_branch" example:"Da Nang Branch"`
	KYCStatus             string       `json:"kyc_status" example:"approved"`
	KYCApprovedAt         *time.Time   `json:"kyc_approved_at,omitempty" example:"2025-11-15T10:30:00Z"`
	KYCApprovedBy         string       `json:"kyc_approved_by,omitempty" example:"admin@paymentgateway.com"`
	KYCRejectedAt         *time.Time   `json:"kyc_rejected_at,omitempty" example:"2025-11-15T10:30:00Z"`
	KYCRejectionReason    string       `json:"kyc_rejection_reason,omitempty" example:"Invalid documents"`
	APIKey                string       `json:"api_key,omitempty" example:"pk_live_abcd1234..."`
	APIKeyLastUsedAt      *time.Time   `json:"api_key_last_used_at,omitempty" example:"2025-11-17T08:15:00Z"`
	WebhookURL            string       `json:"webhook_url,omitempty" example:"https://merchant.com/webhook"`
	WebhookSecret         string       `json:"webhook_secret,omitempty" example:"whsec_abc123..."`
	IsActive              bool         `json:"is_active" example:"true"`
	CreatedAt             time.Time    `json:"created_at" example:"2025-11-10T09:00:00Z"`
	UpdatedAt             time.Time    `json:"updated_at" example:"2025-11-15T10:30:00Z"`
	Balance               *BalanceInfo `json:"balance,omitempty"`
}

// BalanceInfo represents merchant balance information
type BalanceInfo struct {
	AvailableVND     decimal.Decimal `json:"available_vnd" example:"50000000"`
	PendingVND       decimal.Decimal `json:"pending_vnd" example:"5000000"`
	TotalReceivedVND decimal.Decimal `json:"total_received_vnd" example:"100000000"`
	TotalPaidOutVND  decimal.Decimal `json:"total_paid_out_vnd" example:"45000000"`
}

// MerchantListItem represents a merchant in the list view
type MerchantListItem struct {
	ID            string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email         string     `json:"email" example:"merchant@example.com"`
	BusinessName  string     `json:"business_name" example:"Da Nang Beach Resort"`
	KYCStatus     string     `json:"kyc_status" example:"pending"`
	IsActive      bool       `json:"is_active" example:"true"`
	CreatedAt     time.Time  `json:"created_at" example:"2025-11-10T09:00:00Z"`
	KYCApprovedAt *time.Time `json:"kyc_approved_at,omitempty" example:"2025-11-15T10:30:00Z"`
}

// ListMerchantsResponse represents the response for listing merchants
type ListMerchantsResponse struct {
	Merchants []*MerchantListItem `json:"merchants"`
	Total     int                 `json:"total" example:"45"`
	Limit     int                 `json:"limit" example:"20"`
	Offset    int                 `json:"offset" example:"0"`
}

// Admin Payout Management DTOs

// ApprovePayoutRequest represents a request to approve a payout
type ApprovePayoutRequest struct {
	ApprovedBy string `json:"approved_by" binding:"required" example:"admin@paymentgateway.com"`
}

// RejectPayoutRequest represents a request to reject a payout
type RejectPayoutRequest struct {
	Reason string `json:"reason" binding:"required" example:"Insufficient documentation"`
}

// CompletePayoutRequest represents a request to mark a payout as completed
type CompletePayoutRequest struct {
	BankReferenceNumber string `json:"bank_reference_number" binding:"required" example:"VCB20251117001234"`
	ProcessedBy         string `json:"processed_by" binding:"required" example:"admin@paymentgateway.com"`
}

// ListPayoutsQuery represents query parameters for listing payouts
type ListPayoutsQuery struct {
	Status     string `form:"status" binding:"omitempty,oneof=requested approved rejected processing completed failed" example:"requested"`
	MerchantID string `form:"merchant_id" binding:"omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Limit      int    `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
	Offset     int    `form:"offset" binding:"omitempty,min=0" example:"0"`
}

// PayoutDetailResponse represents detailed payout information for admin
type PayoutDetailResponse struct {
	ID                  string          `json:"id" example:"payout_550e8400"`
	MerchantID          string          `json:"merchant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	MerchantName        string          `json:"merchant_name" example:"Da Nang Beach Resort"`
	Amount              decimal.Decimal `json:"amount" example:"10000000"`
	Fee                 decimal.Decimal `json:"fee" example:"50000"`
	NetAmount           decimal.Decimal `json:"net_amount" example:"9950000"`
	Currency            string          `json:"currency" example:"VND"`
	BankAccountName     string          `json:"bank_account_name" example:"NGUYEN VAN A"`
	BankAccountNumber   string          `json:"bank_account_number" example:"1234567890"`
	BankName            string          `json:"bank_name" example:"Vietcombank"`
	BankBranch          string          `json:"bank_branch" example:"Da Nang Branch"`
	Status              string          `json:"status" example:"requested"`
	RequestedAt         time.Time       `json:"requested_at" example:"2025-11-17T08:00:00Z"`
	ApprovedAt          *time.Time      `json:"approved_at,omitempty" example:"2025-11-17T08:30:00Z"`
	ApprovedBy          string          `json:"approved_by,omitempty" example:"admin@paymentgateway.com"`
	RejectedAt          *time.Time      `json:"rejected_at,omitempty" example:"2025-11-17T08:30:00Z"`
	RejectionReason     string          `json:"rejection_reason,omitempty" example:"Insufficient balance"`
	ProcessingAt        *time.Time      `json:"processing_at,omitempty" example:"2025-11-17T09:00:00Z"`
	CompletedAt         *time.Time      `json:"completed_at,omitempty" example:"2025-11-17T10:00:00Z"`
	BankReferenceNumber string          `json:"bank_reference_number,omitempty" example:"VCB20251117001234"`
	FailedAt            *time.Time      `json:"failed_at,omitempty" example:"2025-11-17T10:00:00Z"`
	FailureReason       string          `json:"failure_reason,omitempty" example:"Bank transfer failed"`
	ProcessedBy         string          `json:"processed_by,omitempty" example:"admin@paymentgateway.com"`
	CreatedAt           time.Time       `json:"created_at" example:"2025-11-17T08:00:00Z"`
	UpdatedAt           time.Time       `json:"updated_at" example:"2025-11-17T10:00:00Z"`
}

// AdminPayoutListItem represents a payout in the admin list view (renamed to avoid conflict with payout.go)
type AdminPayoutListItem struct {
	ID           string          `json:"id" example:"payout_550e8400"`
	MerchantID   string          `json:"merchant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	MerchantName string          `json:"merchant_name" example:"Da Nang Beach Resort"`
	Amount       decimal.Decimal `json:"amount" example:"10000000"`
	Currency     string          `json:"currency" example:"VND"`
	Status       string          `json:"status" example:"requested"`
	RequestedAt  time.Time       `json:"requested_at" example:"2025-11-17T08:00:00Z"`
}

// AdminListPayoutsResponse represents the response for listing payouts in admin (renamed to avoid conflict)
type AdminListPayoutsResponse struct {
	Payouts []*AdminPayoutListItem `json:"payouts"`
	Total   int                    `json:"total" example:"15"`
	Limit   int                    `json:"limit" example:"20"`
	Offset  int                    `json:"offset" example:"0"`
}

// Admin System Statistics DTOs

// SystemStatsResponse represents overall system statistics
type SystemStatsResponse struct {
	TotalMerchants      int             `json:"total_merchants" example:"45"`
	ActiveMerchants     int             `json:"active_merchants" example:"40"`
	PendingKYCMerchants int             `json:"pending_kyc_merchants" example:"5"`
	TotalPayments       int             `json:"total_payments" example:"1250"`
	TotalPaymentsVolume decimal.Decimal `json:"total_payments_volume" example:"500000000"`
	TotalPayouts        int             `json:"total_payouts" example:"230"`
	TotalPayoutsVolume  decimal.Decimal `json:"total_payouts_volume" example:"450000000"`
	PendingPayouts      int             `json:"pending_payouts" example:"12"`
	HotWalletBalance    decimal.Decimal `json:"hot_wallet_balance" example:"5000.50"`
	Platform            string          `json:"platform" example:"solana"`
	Timestamp           time.Time       `json:"timestamp" example:"2025-11-17T10:00:00Z"`
}

// DailyStatsQuery represents query parameters for daily statistics
type DailyStatsQuery struct {
	StartDate string `form:"start_date" binding:"required" example:"2025-11-01"`
	EndDate   string `form:"end_date" binding:"required" example:"2025-11-17"`
}

// DailyStatsItem represents statistics for a single day
type DailyStatsItem struct {
	Date           string          `json:"date" example:"2025-11-17"`
	PaymentsCount  int             `json:"payments_count" example:"45"`
	PaymentsVolume decimal.Decimal `json:"payments_volume" example:"25000000"`
	PayoutsCount   int             `json:"payouts_count" example:"8"`
	PayoutsVolume  decimal.Decimal `json:"payouts_volume" example:"20000000"`
	Revenue        decimal.Decimal `json:"revenue" example:"300000"`
}

// DailyStatsResponse represents the response for daily statistics
type DailyStatsResponse struct {
	Stats     []*DailyStatsItem `json:"stats"`
	TotalDays int               `json:"total_days" example:"17"`
	StartDate string            `json:"start_date" example:"2025-11-01"`
	EndDate   string            `json:"end_date" example:"2025-11-17"`
}

// Helper functions to convert models to DTOs

// MerchantToDetailResponse converts a merchant model to a detail response DTO
func MerchantToDetailResponse(merchant *model.Merchant, balance *model.MerchantBalance) *MerchantDetailResponse {
	response := &MerchantDetailResponse{
		ID:            merchant.ID,
		Email:         merchant.Email,
		BusinessName:  merchant.BusinessName,
		OwnerFullName: merchant.OwnerFullName,
		KYCStatus:     string(merchant.KYCStatus),
		IsActive:      merchant.IsActive(),
		CreatedAt:     merchant.CreatedAt,
		UpdatedAt:     merchant.UpdatedAt,
	}

	// Handle nullable string fields
	if merchant.BusinessTaxID.Valid {
		response.BusinessTaxID = merchant.BusinessTaxID.String
	}
	if merchant.BusinessLicenseNumber.Valid {
		response.BusinessLicenseNumber = merchant.BusinessLicenseNumber.String
	}
	if merchant.OwnerIDCard.Valid {
		response.OwnerIDCard = merchant.OwnerIDCard.String
	}
	if merchant.PhoneNumber.Valid {
		response.PhoneNumber = merchant.PhoneNumber.String
	}
	if merchant.BusinessAddress.Valid {
		response.BusinessAddress = merchant.BusinessAddress.String
	}
	if merchant.City.Valid {
		response.City = merchant.City.String
	}
	if merchant.District.Valid {
		response.District = merchant.District.String
	}
	if merchant.BankAccountName.Valid {
		response.BankAccountName = merchant.BankAccountName.String
	}
	if merchant.BankAccountNumber.Valid {
		response.BankAccountNumber = merchant.BankAccountNumber.String
	}
	if merchant.BankName.Valid {
		response.BankName = merchant.BankName.String
	}
	if merchant.BankBranch.Valid {
		response.BankBranch = merchant.BankBranch.String
	}
	if merchant.KYCApprovedAt.Valid {
		response.KYCApprovedAt = &merchant.KYCApprovedAt.Time
	}
	if merchant.KYCApprovedBy.Valid {
		response.KYCApprovedBy = merchant.KYCApprovedBy.String
	}
	// Note: KYCRejectedAt field doesn't exist in model, only KYCRejectionReason
	if merchant.KYCRejectionReason.Valid {
		response.KYCRejectionReason = merchant.KYCRejectionReason.String
	}
	if merchant.APIKey.Valid {
		response.APIKey = merchant.APIKey.String
	}
	// Note: APIKeyLastUsedAt field doesn't exist in current model
	if merchant.WebhookURL.Valid {
		response.WebhookURL = merchant.WebhookURL.String
	}
	if merchant.WebhookSecret.Valid {
		response.WebhookSecret = merchant.WebhookSecret.String
	}

	if balance != nil {
		response.Balance = &BalanceInfo{
			AvailableVND:     balance.AvailableVND,
			PendingVND:       balance.PendingVND,
			TotalReceivedVND: balance.TotalReceivedVND,
			TotalPaidOutVND:  balance.TotalPaidOutVND,
		}
	}

	return response
}

// MerchantToListItem converts a merchant model to a list item DTO
func MerchantToListItem(merchant *model.Merchant) *MerchantListItem {
	item := &MerchantListItem{
		ID:           merchant.ID,
		Email:        merchant.Email,
		BusinessName: merchant.BusinessName,
		KYCStatus:    string(merchant.KYCStatus),
		IsActive:     merchant.IsActive(),
		CreatedAt:    merchant.CreatedAt,
	}

	// Handle nullable KYCApprovedAt
	if merchant.KYCApprovedAt.Valid {
		item.KYCApprovedAt = &merchant.KYCApprovedAt.Time
	}

	return item
}

// PayoutToDetailResponse converts a payout model to a detail response DTO
func PayoutToDetailResponse(payout *model.Payout, merchantName string) *PayoutDetailResponse {
	response := &PayoutDetailResponse{
		ID:                payout.ID,
		MerchantID:        payout.MerchantID,
		MerchantName:      merchantName,
		Amount:            payout.AmountVND,
		Fee:               payout.FeeVND,
		NetAmount:         payout.NetAmountVND,
		Currency:          "VND",
		BankAccountName:   payout.BankAccountName,
		BankAccountNumber: payout.BankAccountNumber,
		BankName:          payout.BankName,
		Status:            string(payout.Status),
		RequestedAt:       payout.CreatedAt,
		CreatedAt:         payout.CreatedAt,
		UpdatedAt:         payout.UpdatedAt,
	}

	// Handle nullable fields
	if payout.BankBranch.Valid {
		response.BankBranch = payout.BankBranch.String
	}
	if payout.ApprovedAt.Valid {
		response.ApprovedAt = &payout.ApprovedAt.Time
	}
	if payout.ApprovedBy.Valid {
		response.ApprovedBy = payout.ApprovedBy.String
	}
	if payout.RejectionReason.Valid {
		response.RejectionReason = payout.RejectionReason.String
	}
	if payout.ProcessedBy.Valid {
		response.ProcessedBy = payout.ProcessedBy.String
	}
	if payout.ProcessedAt.Valid {
		response.ProcessingAt = &payout.ProcessedAt.Time
	}
	if payout.CompletionDate.Valid {
		response.CompletedAt = &payout.CompletionDate.Time
	}
	if payout.BankReferenceNumber.Valid {
		response.BankReferenceNumber = payout.BankReferenceNumber.String
	}
	if payout.FailureReason.Valid {
		response.FailureReason = payout.FailureReason.String
	}

	return response
}

// AdminPayoutToListItem converts a payout model to an admin list item DTO (renamed to avoid conflict)
func AdminPayoutToListItem(payout *model.Payout, merchantName string) *AdminPayoutListItem {
	return &AdminPayoutListItem{
		ID:           payout.ID,
		MerchantID:   payout.MerchantID,
		MerchantName: merchantName,
		Amount:       payout.AmountVND,
		Currency:     "VND",
		Status:       string(payout.Status),
		RequestedAt:  payout.CreatedAt,
	}
}

// Admin Compliance Management DTOs

// GetTravelRuleDataQuery represents query parameters for Travel Rule data
type GetTravelRuleDataQuery struct {
	StartDate string `form:"start_date" binding:"required" example:"2025-11-01"`
	EndDate   string `form:"end_date" binding:"required" example:"2025-11-17"`
	Country   string `form:"country" binding:"omitempty" example:"VN"` // ISO 3166-1 alpha-2
	Limit     int    `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
	Offset    int    `form:"offset" binding:"omitempty,min=0" example:"0"`
}

// TravelRuleDataItem represents a Travel Rule data item
type TravelRuleDataItem struct {
	ID                  string          `json:"id"`
	PaymentID           string          `json:"payment_id"`
	PayerFullName       string          `json:"payer_full_name"`
	PayerWalletAddress  string          `json:"payer_wallet_address"`
	PayerCountry        string          `json:"payer_country"`
	PayerIDDocument     *string         `json:"payer_id_document,omitempty"`
	MerchantFullName    string          `json:"merchant_full_name"`
	MerchantCountry     string          `json:"merchant_country"`
	TransactionAmount   decimal.Decimal `json:"transaction_amount"`
	TransactionCurrency string          `json:"transaction_currency"`
	CreatedAt           time.Time       `json:"created_at"`
}

// GetTravelRuleDataResponse represents the response for Travel Rule data
type GetTravelRuleDataResponse struct {
	Data   []TravelRuleDataItem `json:"data"`
	Total  int                  `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}

// ApproveKYCDocumentRequest represents a request to approve a KYC document
type ApproveKYCDocumentRequest struct {
	ApprovedBy string `json:"approved_by" binding:"required" example:"admin@paymentgateway.com"`
	Notes      string `json:"notes,omitempty" validate:"omitempty,max=500"`
}

// RejectKYCDocumentRequest represents a request to reject a KYC document
type RejectKYCDocumentRequest struct {
	RejectedBy string `json:"rejected_by" binding:"required" example:"admin@paymentgateway.com"`
	Reason     string `json:"reason" binding:"required,max=500" example:"Document is not clear"`
}

// UpgradeMerchantTierRequest represents a request to upgrade a merchant's KYC tier
type UpgradeMerchantTierRequest struct {
	Tier string `json:"tier" binding:"required,oneof=tier1 tier2 tier3" example:"tier2"`
}

// GetComplianceMetricsResponse represents compliance metrics for a merchant
type GetComplianceMetricsResponse struct {
	MerchantID             string          `json:"merchant_id"`
	KYCTier                string          `json:"kyc_tier"`
	MonthlyLimitUSD        decimal.Decimal `json:"monthly_limit_usd"`
	MonthlyVolumeUSD       decimal.Decimal `json:"monthly_volume_usd"`
	RemainingLimitUSD      decimal.Decimal `json:"remaining_limit_usd"`
	UtilizationPercent     float64         `json:"utilization_percent"`
	TravelRuleTransactions int64           `json:"travel_rule_transactions"`
	TotalTransactions      int64           `json:"total_transactions"`
	LastScreeningDate      *time.Time      `json:"last_screening_date,omitempty"`
}

// TravelRuleDataToListItem converts a model.TravelRuleData to TravelRuleDataItem
func TravelRuleDataToListItem(data *model.TravelRuleData) TravelRuleDataItem {
	item := TravelRuleDataItem{
		ID:                  data.ID, // ID is already a string
		PaymentID:           data.PaymentID,
		PayerFullName:       data.PayerFullName,
		PayerWalletAddress:  data.PayerWalletAddress,
		PayerCountry:        data.PayerCountry,
		MerchantFullName:    data.MerchantFullName,
		MerchantCountry:     data.MerchantCountry,
		TransactionAmount:   data.TransactionAmount,
		TransactionCurrency: data.TransactionCurrency,
		CreatedAt:           data.CreatedAt,
	}

	// Handle optional PayerIDDocument
	if data.PayerIDDocument.Valid {
		idDoc := data.PayerIDDocument.String
		item.PayerIDDocument = &idDoc
	}

	return item
}
