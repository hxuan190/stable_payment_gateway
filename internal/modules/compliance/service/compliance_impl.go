package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/audit/domain"
	auditRepository "github.com/hxuan190/stable_payment_gateway/internal/modules/audit/repository"
	complianceDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/domain"
	complianceRepository "github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/repository"
	infrastructureRepository "github.com/hxuan190/stable_payment_gateway/internal/modules/infrastructure/repository"
	merchantDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/domain"
	merchantRepository "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/repository"
	paymentDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
)

var (
	// ErrMonthlyLimitExceeded is returned when merchant exceeds their monthly limit
	ErrMonthlyLimitExceeded = errors.New("monthly limit exceeded")
	// ErrTravelRuleRequired is returned when Travel Rule data is required but not provided
	ErrTravelRuleRequired = errors.New("Travel Rule data required for transactions > $1000 USD")
	// ErrInsufficientKYCDocuments is returned when merchant doesn't have required KYC documents
	ErrInsufficientKYCDocuments = errors.New("insufficient KYC documents for requested tier")
	// ErrKYCTierUpgradeNotAllowed is returned when tier upgrade is not allowed
	ErrKYCTierUpgradeNotAllowed = errors.New("KYC tier upgrade not allowed")
)

// ComplianceService handles compliance-related operations
type ComplianceService interface {
	// Payment Compliance
	ValidatePaymentCompliance(ctx context.Context, payment *paymentDomain.Payment, travelRuleData *complianceDomain.TravelRuleData, fromAddress string) error
	CheckMonthlyLimit(ctx context.Context, merchantID string, amountUSD decimal.Decimal) error
	StoreTravelRuleData(ctx context.Context, data *complianceDomain.TravelRuleData) error

	// KYC Management
	UpgradeKYCTier(ctx context.Context, merchantID string, tier string) error
	CheckKYCTierRequirements(ctx context.Context, merchantID string, targetTier string) error
	GetKYCDocuments(ctx context.Context, merchantID string) ([]*merchantDomain.KYCDocument, error)

	// Reporting
	GetTravelRuleReport(ctx context.Context, startDate, endDate time.Time) ([]*complianceDomain.TravelRuleData, error)
	GetComplianceMetrics(ctx context.Context, merchantID string) (*ComplianceMetrics, error)
}

// ComplianceMetrics represents compliance metrics for a merchant
type ComplianceMetrics struct {
	MerchantID             string
	KYCTier                string
	MonthlyLimitUSD        decimal.Decimal
	MonthlyVolumeUSD       decimal.Decimal
	RemainingLimitUSD      decimal.Decimal
	UtilizationPercent     float64
	TravelRuleTransactions int64
	TotalTransactions      int64
	LastScreeningDate      *time.Time
}

type complianceServiceImpl struct {
	amlService      AMLService
	merchantRepo    *merchantRepository.MerchantRepository
	travelRuleRepo  complianceRepository.TravelRuleRepository
	kycDocumentRepo infrastructureRepository.KYCDocumentRepository
	paymentRepo     paymentDomain.PaymentRepository
	auditRepo       *auditRepository.AuditRepository
	logger          *logger.Logger
}

// NewComplianceService creates a new compliance service
func NewComplianceService(
	amlService AMLService,
	merchantRepo *merchantRepository.MerchantRepository,
	travelRuleRepo complianceRepository.TravelRuleRepository,
	kycDocumentRepo infrastructureRepository.KYCDocumentRepository,
	paymentRepo paymentDomain.PaymentRepository,
	auditRepo *auditRepository.AuditRepository,
	logger *logger.Logger,
) ComplianceService {
	return &complianceServiceImpl{
		amlService:      amlService,
		merchantRepo:    merchantRepo,
		travelRuleRepo:  travelRuleRepo,
		kycDocumentRepo: kycDocumentRepo,
		paymentRepo:     paymentRepo,
		auditRepo:       auditRepo,
		logger:          logger,
	}
}

// ValidatePaymentCompliance performs comprehensive compliance validation on a payment
func (s *complianceServiceImpl) ValidatePaymentCompliance(
	ctx context.Context,
	payment *paymentDomain.Payment,
	travelRuleData *complianceDomain.TravelRuleData,
	fromAddress string,
) error {
	// Fixed: Use AmountCrypto as USD equivalent since we use USDT/USDC (USD-pegged stablecoins)
	amountUSD := payment.AmountCrypto

	s.logger.Info("Validating payment compliance", map[string]interface{}{
		"payment_id":  payment.ID,
		"merchant_id": payment.MerchantID,
		"amount_usd":  amountUSD.String(),
	})

	// Step 1: Check monthly limit
	err := s.CheckMonthlyLimit(ctx, payment.MerchantID, amountUSD)
	if err != nil {
		s.logger.Warn("Monthly limit check failed", map[string]interface{}{
			"payment_id":  payment.ID,
			"merchant_id": payment.MerchantID,
			"error":       err.Error(),
		})
		return err
	}

	// Step 2: Check Travel Rule requirement (transactions > $1000)
	travelRuleThreshold := decimal.NewFromInt(1000)
	if amountUSD.GreaterThan(travelRuleThreshold) {
		if travelRuleData == nil {
			s.logger.Warn("Travel Rule data missing for high-value transaction", map[string]interface{}{
				"payment_id": payment.ID,
				"amount_usd": amountUSD.String(),
			})
			return ErrTravelRuleRequired
		}

		// Store Travel Rule data
		err := s.StoreTravelRuleData(ctx, travelRuleData)
		if err != nil {
			return fmt.Errorf("failed to store Travel Rule data: %w", err)
		}
	}

	// Step 3: AML screening (if from_address is provided)
	if fromAddress != "" {
		paymentID, err := uuid.Parse(payment.ID)
		if err != nil {
			return fmt.Errorf("invalid payment ID: %w", err)
		}

		err = s.amlService.ValidateTransaction(ctx, paymentID, fromAddress, string(payment.Chain), payment.AmountCrypto)
		if err != nil {
			s.logger.Warn("AML validation failed", map[string]interface{}{
				"payment_id":   payment.ID,
				"from_address": fromAddress,
				"error":        err.Error(),
			})
			return fmt.Errorf("AML validation failed: %w", err)
		}
	}

	s.logger.Info("Payment compliance validation passed", map[string]interface{}{
		"payment_id":  payment.ID,
		"merchant_id": payment.MerchantID,
	})

	return nil
}

// CheckMonthlyLimit checks if the merchant is within their monthly transaction limit
func (s *complianceServiceImpl) CheckMonthlyLimit(ctx context.Context, merchantID string, amountUSD decimal.Decimal) error {
	merchant, err := s.merchantRepo.GetByID(merchantID)
	if err != nil {
		return fmt.Errorf("failed to get merchant: %w", err)
	}

	// Check if we need to reset the monthly volume (new month)
	now := time.Now()
	if merchant.VolumeLastResetAt.Month() != now.Month() || merchant.VolumeLastResetAt.Year() != now.Year() {
		// Reset volume for new month
		merchant.TotalVolumeThisMonthUSD = decimal.Zero
		merchant.VolumeLastResetAt = now

		// Update merchant in database
		err = s.merchantRepo.Update(merchant)
		if err != nil {
			s.logger.Error("Failed to reset monthly volume", err, map[string]interface{}{
				"merchant_id": merchantID,
			})
			// Continue anyway - don't fail the check
		}
	}

	// Calculate new total if this payment is approved
	newTotal := merchant.TotalVolumeThisMonthUSD.Add(amountUSD)

	// Check if it exceeds the limit
	if newTotal.GreaterThan(merchant.MonthlyLimitUSD) {
		remaining := merchant.MonthlyLimitUSD.Sub(merchant.TotalVolumeThisMonthUSD)
		s.logger.Warn("Monthly limit would be exceeded", map[string]interface{}{
			"merchant_id":      merchantID,
			"current_volume":   merchant.TotalVolumeThisMonthUSD.String(),
			"monthly_limit":    merchant.MonthlyLimitUSD.String(),
			"requested_amount": amountUSD.String(),
			"remaining_limit":  remaining.String(),
		})
		return ErrMonthlyLimitExceeded
	}

	return nil
}

// StoreTravelRuleData stores Travel Rule data for a transaction
func (s *complianceServiceImpl) StoreTravelRuleData(ctx context.Context, data *complianceDomain.TravelRuleData) error {
	if data == nil {
		return errors.New("travel rule data cannot be nil")
	}

	// Validate that amount meets threshold
	if !data.IsValidForThreshold() {
		return errors.New("transaction amount does not meet Travel Rule threshold ($1000 USD)")
	}

	// Store in database
	err := s.travelRuleRepo.Create(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to create Travel Rule record: %w", err)
	}

	// Log audit trail
	auditLog := &domain.AuditLog{
		ID:             uuid.New().String(), // Fixed: convert UUID to string
		ActorType:      domain.ActorTypeSystem,
		Action:         "travel_rule_stored",
		ActionCategory: domain.ActionCategoryPayment,
		ResourceType:   "payment",
		ResourceID:     data.PaymentID, // Fixed: PaymentID is already a string
		Status:         domain.AuditStatusSuccess,
		Metadata: map[string]interface{}{
			"payer_country":      data.PayerCountry,
			"merchant_country":   data.MerchantCountry,
			"transaction_amount": data.TransactionAmount.String(),
			"is_cross_border":    data.IsCrossBorder(),
		},
		CreatedAt: time.Now(),
	}

	if err := s.auditRepo.Create(auditLog); err != nil { // Fixed: removed ctx parameter
		s.logger.Error("Failed to create audit log for Travel Rule", err, nil)
		// Don't fail the operation if audit logging fails
	}

	s.logger.Info("Travel Rule data stored", map[string]interface{}{
		"payment_id":       data.PaymentID,
		"payer_country":    data.PayerCountry,
		"merchant_country": data.MerchantCountry,
	})

	return nil
}

// UpgradeKYCTier upgrades a merchant's KYC tier
func (s *complianceServiceImpl) UpgradeKYCTier(ctx context.Context, merchantID string, tier string) error {
	// Validate tier value
	if tier != "tier1" && tier != "tier2" && tier != "tier3" {
		return errors.New("invalid KYC tier: must be tier1, tier2, or tier3")
	}

	// Check if merchant meets requirements for this tier
	err := s.CheckKYCTierRequirements(ctx, merchantID, tier)
	if err != nil {
		return err
	}

	// Get merchant
	merchant, err := s.merchantRepo.GetByID(merchantID)
	if err != nil {
		return fmt.Errorf("failed to get merchant: %w", err)
	}

	oldTier := merchant.KYCTier

	// Update tier and monthly limit
	merchant.KYCTier = merchantDomain.KYCTier(tier) // Fixed: convert string to KYCTier type
	switch tier {
	case "tier1":
		merchant.MonthlyLimitUSD = decimal.NewFromInt(5000)
	case "tier2":
		merchant.MonthlyLimitUSD = decimal.NewFromInt(50000)
	case "tier3":
		merchant.MonthlyLimitUSD = decimal.NewFromInt(999999999) // Unlimited (use very high value)
	}

	// Save to database
	err = s.merchantRepo.Update(merchant)
	if err != nil {
		return fmt.Errorf("failed to update merchant: %w", err)
	}

	// Log audit trail
	auditLog := &domain.AuditLog{
		ID:             uuid.New().String(), // Fixed: convert UUID to string
		ActorType:      domain.ActorTypeAdmin,
		Action:         "kyc_tier_upgraded",
		ActionCategory: domain.ActionCategoryKYC,
		ResourceType:   "merchant",
		ResourceID:     merchantID, // Fixed: merchantID is already a string
		Status:         domain.AuditStatusSuccess,
		Metadata: map[string]interface{}{
			"old_tier":          string(oldTier), // Fixed: convert KYCTier to string
			"new_tier":          tier,
			"new_monthly_limit": merchant.MonthlyLimitUSD.String(),
		},
		CreatedAt: time.Now(),
	}

	if err := s.auditRepo.Create(auditLog); err != nil { // Fixed: removed ctx parameter
		s.logger.Error("Failed to create audit log for tier upgrade", err, nil)
	}

	s.logger.Info("KYC tier upgraded", map[string]interface{}{
		"merchant_id": merchantID,
		"old_tier":    oldTier,
		"new_tier":    tier,
	})

	return nil
}

// CheckKYCTierRequirements checks if a merchant meets the requirements for a specific tier
func (s *complianceServiceImpl) CheckKYCTierRequirements(ctx context.Context, merchantID string, targetTier string) error {
	switch targetTier {
	case "tier1":
		// Tier 1 has no special requirements (email + phone is enough)
		return nil

	case "tier2":
		// Tier 2 requires business registration document
		hasBusinessReg, err := s.kycDocumentRepo.HasApprovedDocumentOfType(ctx, merchantID, merchantDomain.KYCDocumentTypeBusinessRegistration)
		if err != nil {
			return fmt.Errorf("failed to check business registration: %w", err)
		}
		if !hasBusinessReg {
			return fmt.Errorf("%w: business registration document required for Tier 2", ErrInsufficientKYCDocuments)
		}
		return nil

	case "tier3":
		// Tier 3 requires: business registration + tax certificate + bank statement
		requiredDocs := []merchantDomain.KYCDocumentType{
			merchantDomain.KYCDocumentTypeBusinessRegistration,
			merchantDomain.KYCDocumentTypeTaxCertificate,
			merchantDomain.KYCDocumentTypeBankStatement,
		}

		for _, docType := range requiredDocs {
			hasDoc, err := s.kycDocumentRepo.HasApprovedDocumentOfType(ctx, merchantID, docType)
			if err != nil {
				return fmt.Errorf("failed to check %s: %w", docType, err)
			}
			if !hasDoc {
				return fmt.Errorf("%w: %s required for Tier 3", ErrInsufficientKYCDocuments, docType)
			}
		}
		return nil

	default:
		return errors.New("invalid tier")
	}
}

// GetKYCDocuments retrieves all KYC documents for a merchant
func (s *complianceServiceImpl) GetKYCDocuments(ctx context.Context, merchantID string) ([]*merchantDomain.KYCDocument, error) {
	documents, err := s.kycDocumentRepo.GetByMerchantID(ctx, merchantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get KYC documents: %w", err)
	}
	return documents, nil
}

// GetTravelRuleReport generates a Travel Rule compliance report
func (s *complianceServiceImpl) GetTravelRuleReport(ctx context.Context, startDate, endDate time.Time) ([]*complianceDomain.TravelRuleData, error) {
	filter := complianceRepository.TravelRuleFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
		MinAmount: floatPtr(1000), // Travel Rule threshold
	}

	data, err := s.travelRuleRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get Travel Rule data: %w", err)
	}

	return data, nil
}

// GetComplianceMetrics retrieves compliance metrics for a merchant
func (s *complianceServiceImpl) GetComplianceMetrics(ctx context.Context, merchantID string) (*ComplianceMetrics, error) {
	// Get merchant info
	merchant, err := s.merchantRepo.GetByID(merchantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merchant: %w", err)
	}

	// Calculate remaining limit
	remaining := merchant.MonthlyLimitUSD.Sub(merchant.TotalVolumeThisMonthUSD)
	if remaining.LessThan(decimal.Zero) {
		remaining = decimal.Zero
	}

	// Calculate utilization percentage
	var utilization float64
	if merchant.MonthlyLimitUSD.GreaterThan(decimal.Zero) {
		utilization, _ = merchant.TotalVolumeThisMonthUSD.Div(merchant.MonthlyLimitUSD).Mul(decimal.NewFromInt(100)).Float64()
	}

	// TODO: Get transaction counts (requires payment repository query)
	// For now, set to 0
	travelRuleCount := int64(0)
	totalCount := int64(0)

	metrics := &ComplianceMetrics{
		MerchantID:             merchantID,
		KYCTier:                string(merchant.KYCTier), // Fixed: convert KYCTier to string
		MonthlyLimitUSD:        merchant.MonthlyLimitUSD,
		MonthlyVolumeUSD:       merchant.TotalVolumeThisMonthUSD,
		RemainingLimitUSD:      remaining,
		UtilizationPercent:     utilization,
		TravelRuleTransactions: travelRuleCount,
		TotalTransactions:      totalCount,
		LastScreeningDate:      nil, // TODO: Implement
	}

	return metrics, nil
}

// Helper function
func floatPtr(f float64) *float64 {
	return &f
}
