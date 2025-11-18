package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/trmlabs"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
)

var (
	// ErrSanctionedAddress is returned when a wallet address is on sanctions list
	ErrSanctionedAddress = errors.New("wallet address is sanctioned")
	// ErrHighRiskAddress is returned when a wallet address has high risk score
	ErrHighRiskAddress = errors.New("wallet address has high risk score")
	// ErrBlockedFlag is returned when address has a blocked flag
	ErrBlockedFlag = errors.New("wallet address has blocked flag")
	// ErrAMLScreeningFailed is returned when AML screening fails
	ErrAMLScreeningFailed = errors.New("AML screening failed")
)

// AMLScreeningResult represents the result of an AML screening
type AMLScreeningResult struct {
	WalletAddress string
	Chain         string
	RiskScore     int      // 0-100
	IsSanctioned  bool
	Flags         []string
	Details       map[string]string
	ScreenedAt    time.Time
}

// AMLService handles AML (Anti-Money Laundering) screening operations
type AMLService interface {
	ScreenWalletAddress(ctx context.Context, address string, chain string) (*AMLScreeningResult, error)
	RecordScreeningResult(ctx context.Context, paymentID uuid.UUID, result *AMLScreeningResult) error
	ValidateTransaction(ctx context.Context, paymentID uuid.UUID, fromAddress string, chain string, amount decimal.Decimal) error
	GetScreeningHistory(ctx context.Context, walletAddress string) ([]*AMLScreeningResult, error)
}

type amlServiceImpl struct {
	trmClient  TRMLabsClient
	auditRepo  *repository.AuditRepository  // Fixed: correct repository type
	paymentRepo repository.PaymentRepository
	logger     *logger.Logger
	// Risk thresholds
	riskScoreThreshold int  // Reject if risk score >= this value
	autoRejectSanctioned bool // Auto-reject sanctioned addresses
}

// TRMLabsClient is an interface for TRM Labs screening client
// This allows us to easily swap between real and mock implementations
type TRMLabsClient interface {
	ScreenAddress(ctx context.Context, address string, chain string) (*trmlabs.ScreeningResponse, error)
	BatchScreenAddresses(ctx context.Context, addresses []string, chain string) (map[string]*trmlabs.ScreeningResponse, error)
}

// NewAMLService creates a new AML service
func NewAMLService(
	trmClient TRMLabsClient,
	auditRepo *repository.AuditRepository,  // Fixed: correct repository type
	paymentRepo repository.PaymentRepository,
	logger *logger.Logger,
) AMLService {
	return &amlServiceImpl{
		trmClient:  trmClient,
		auditRepo:  auditRepo,
		paymentRepo: paymentRepo,
		logger:     logger,
		riskScoreThreshold: 80, // Default: reject if risk score >= 80
		autoRejectSanctioned: true, // Default: auto-reject sanctioned addresses
	}
}

// ScreenWalletAddress screens a wallet address against AML database
func (s *amlServiceImpl) ScreenWalletAddress(ctx context.Context, address string, chain string) (*AMLScreeningResult, error) {
	s.logger.Info("Screening wallet address", map[string]interface{}{
		"address": address,
		"chain":   chain,
	})

	// Call TRM Labs API
	trmResult, err := s.trmClient.ScreenAddress(ctx, address, chain)
	if err != nil {
		s.logger.Error("Failed to screen address with TRM Labs", err, map[string]interface{}{
			"address": address,
			"chain":   chain,
		})
		return nil, fmt.Errorf("failed to screen address: %w", err)
	}

	// Convert to our internal format
	result := &AMLScreeningResult{
		WalletAddress: trmResult.Address,
		Chain:         trmResult.Chain,
		RiskScore:     trmResult.RiskScore,
		IsSanctioned:  trmResult.IsSanctioned,
		Flags:         trmResult.Flags,
		Details:       trmResult.Details,
		ScreenedAt:    trmResult.ScreenedAt,
	}

	s.logger.Info("Wallet address screening completed", map[string]interface{}{
		"address":      address,
		"chain":        chain,
		"risk_score":   result.RiskScore,
		"is_sanctioned": result.IsSanctioned,
		"flags":        result.Flags,
	})

	return result, nil
}

// RecordScreeningResult records an AML screening result in the audit log
func (s *amlServiceImpl) RecordScreeningResult(ctx context.Context, paymentID uuid.UUID, result *AMLScreeningResult) error {
	metadata := map[string]interface{}{
		"wallet_address": result.WalletAddress,
		"chain":          result.Chain,
		"risk_score":     result.RiskScore,
		"is_sanctioned":  result.IsSanctioned,
		"flags":          result.Flags,
		"details":        result.Details,
		"screened_at":    result.ScreenedAt,
	}

	auditLog := &model.AuditLog{
		ID:             uuid.New().String(),  // Fixed: convert UUID to string
		ActorType:      model.ActorTypeSystem,
		Action:         "aml_screening",
		ActionCategory: model.ActionCategoryPayment,
		ResourceType:   "payment",
		ResourceID:     paymentID.String(),  // Fixed: convert UUID to string
		Status:         model.AuditStatusSuccess,
		Metadata:       metadata,
		CreatedAt:      time.Now(),
	}

	if err := s.auditRepo.Create(auditLog); err != nil {  // Fixed: removed ctx parameter
		s.logger.Error("Failed to record AML screening result", err, map[string]interface{}{
			"payment_id": paymentID,
		})
		return fmt.Errorf("failed to record screening result: %w", err)
	}

	return nil
}

// ValidateTransaction performs comprehensive AML validation on a transaction
// This is the main entry point for AML checks during payment processing
func (s *amlServiceImpl) ValidateTransaction(ctx context.Context, paymentID uuid.UUID, fromAddress string, chain string, amount decimal.Decimal) error {
	s.logger.Info("Validating transaction for AML compliance", map[string]interface{}{
		"payment_id":   paymentID,
		"from_address": fromAddress,
		"chain":        chain,
		"amount":       amount.String(),
	})

	// Step 1: Screen the wallet address
	result, err := s.ScreenWalletAddress(ctx, fromAddress, chain)
	if err != nil {
		return fmt.Errorf("AML screening failed: %w", err)
	}

	// Step 2: Record the screening result
	if err := s.RecordScreeningResult(ctx, paymentID, result); err != nil {
		s.logger.Error("Failed to record screening result", err, nil)
		// Don't fail the transaction if we can't record, just log
	}

	// Step 3: Check if address is sanctioned
	if result.IsSanctioned && s.autoRejectSanctioned {
		s.logger.Warn("Transaction rejected: sanctioned address", map[string]interface{}{
			"payment_id":   paymentID,
			"from_address": fromAddress,
			"flags":        result.Flags,
		})
		return ErrSanctionedAddress
	}

	// Step 4: Check risk score threshold
	if result.RiskScore >= s.riskScoreThreshold {
		s.logger.Warn("Transaction rejected: high risk score", map[string]interface{}{
			"payment_id":   paymentID,
			"from_address": fromAddress,
			"risk_score":   result.RiskScore,
			"threshold":    s.riskScoreThreshold,
			"flags":        result.Flags,
		})
		return ErrHighRiskAddress
	}

	// Step 5: Check for specific high-risk flags
	for _, flag := range result.Flags {
		if isBlockedFlag(flag) {
			s.logger.Warn("Transaction rejected: blocked flag", map[string]interface{}{
				"payment_id":   paymentID,
				"from_address": fromAddress,
				"flag":         flag,
			})
			return ErrBlockedFlag
		}
	}

	s.logger.Info("Transaction passed AML validation", map[string]interface{}{
		"payment_id":   paymentID,
		"from_address": fromAddress,
		"risk_score":   result.RiskScore,
	})

	return nil
}

// GetScreeningHistory retrieves the screening history for a wallet address
func (s *amlServiceImpl) GetScreeningHistory(ctx context.Context, walletAddress string) ([]*AMLScreeningResult, error) {
	// Query audit logs for AML screening records
	// This is a simplified implementation - in production, you might want a dedicated table
	action := "aml_screening"
	resourceType := "payment"
	auditLogs, err := s.auditRepo.List(repository.AuditFilter{  // Fixed: removed ctx parameter
		Action:       &action,
		ResourceType: &resourceType,
		Limit:        100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve screening history: %w", err)
	}

	var results []*AMLScreeningResult
	for _, log := range auditLogs {
		if addr, ok := log.Metadata["wallet_address"].(string); ok && addr == walletAddress {
			result := &AMLScreeningResult{
				WalletAddress: addr,
			}

			// Parse metadata
			if chain, ok := log.Metadata["chain"].(string); ok {
				result.Chain = chain
			}
			if riskScore, ok := log.Metadata["risk_score"].(float64); ok {
				result.RiskScore = int(riskScore)
			}
			if isSanctioned, ok := log.Metadata["is_sanctioned"].(bool); ok {
				result.IsSanctioned = isSanctioned
			}
			if flags, ok := log.Metadata["flags"].([]interface{}); ok {
				for _, f := range flags {
					if flagStr, ok := f.(string); ok {
						result.Flags = append(result.Flags, flagStr)
					}
				}
			}

			result.ScreenedAt = log.CreatedAt
			results = append(results, result)
		}
	}

	return results, nil
}

// isBlockedFlag checks if a flag should result in automatic rejection
func isBlockedFlag(flag string) bool {
	blockedFlags := []string{
		"sanctions",
		"ofac",
		"terrorist-financing",
		"child-exploitation",
	}

	for _, blocked := range blockedFlags {
		if flag == blocked {
			return true
		}
	}

	return false
}
