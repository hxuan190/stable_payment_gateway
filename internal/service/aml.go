package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	paymentdomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/trmlabs"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
)

var (
	// ErrSanctionedAddress is returned when a wallet address is on sanctions list
	ErrSanctionedAddress = errors.New("wallet address is sanctioned")
	// ErrRiskCritical is returned when risk score >= 86 (block transaction)
	ErrRiskCritical = errors.New("wallet address has critical risk score (>= 86)")
	// ErrRiskHigh is returned when risk score 61-85 (hold for manual review)
	ErrRiskHigh = errors.New("wallet address has high risk score (61-85), manual review required")
	// ErrHighRiskAddress is returned when a wallet address has high risk score (deprecated, use ErrRiskCritical/ErrRiskHigh)
	ErrHighRiskAddress = errors.New("wallet address has high risk score")
	// ErrBlockedFlag is returned when address has a blocked flag
	ErrBlockedFlag = errors.New("wallet address has blocked flag")
	// ErrAMLScreeningFailed is returned when AML screening fails
	ErrAMLScreeningFailed = errors.New("AML screening failed")
	// ErrVelocityLimitExceeded is returned when wallet exceeds transaction velocity limit
	ErrVelocityLimitExceeded = errors.New("velocity limit exceeded: too many transactions from this address")
)

const (
	// Risk score thresholds based on Kill Switch model
	RiskScoreCriticalThreshold = 86
	RiskScoreHighThreshold     = 61
	RiskScoreMediumThreshold   = 20

	// Cache TTL for wallet screening results (24 hours)
	WalletScreeningCacheTTL = 24 * time.Hour

	// Base metadata risk score
	BaseMetadataRisk = 10
)

// AMLScreeningResult represents the result of an AML screening
type AMLScreeningResult struct {
	WalletAddress string
	Chain         string
	RiskScore     int // 0-100
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
	trmClient            TRMLabsClient
	auditRepo            *repository.AuditRepository
	paymentRepo          paymentdomain.PaymentRepository
	ruleEngine           *RuleEngine
	redisClient          *redis.Client
	logger               *logger.Logger
	riskScoreThreshold   int
	autoRejectSanctioned bool
}

// TRMLabsClient is an interface for TRM Labs screening client
// This allows us to easily swap between real and mock implementations
type TRMLabsClient interface {
	ScreenAddress(ctx context.Context, address string, chain string) (*trmlabs.ScreeningResponse, error)
	BatchScreenAddresses(ctx context.Context, addresses []string, chain string) (map[string]*trmlabs.ScreeningResponse, error)
}

func NewAMLService(
	trmClient TRMLabsClient,
	auditRepo *repository.AuditRepository,
	paymentRepo paymentdomain.PaymentRepository,
	ruleEngine *RuleEngine,
	redisClient *redis.Client,
	logger *logger.Logger,
) AMLService {
	return &amlServiceImpl{
		trmClient:            trmClient,
		auditRepo:            auditRepo,
		paymentRepo:          paymentRepo,
		ruleEngine:           ruleEngine,
		redisClient:          redisClient,
		logger:               logger,
		riskScoreThreshold:   RiskScoreCriticalThreshold,
		autoRejectSanctioned: true,
	}
}

// ScreenWalletAddress screens a wallet address against AML database with Redis caching
func (s *amlServiceImpl) ScreenWalletAddress(ctx context.Context, address string, chain string) (*AMLScreeningResult, error) {
	s.logger.Info("Screening wallet address", map[string]interface{}{
		"address": address,
		"chain":   chain,
	})

	// TASK 1: CHECK REDIS CACHE FIRST
	cacheKey := fmt.Sprintf("aml:screen:%s:%s", chain, address)

	// Try to get from cache
	cachedData, err := s.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		// Cache hit - deserialize and return
		var result AMLScreeningResult
		if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
			s.logger.Info("Wallet screening result retrieved from cache", map[string]interface{}{
				"address":    address,
				"chain":      chain,
				"risk_score": result.RiskScore,
				"cache_hit":  true,
			})
			return &result, nil
		}
		// If unmarshal fails, log and continue to API call
		s.logger.Warn("Failed to unmarshal cached screening result, fetching from API", map[string]interface{}{
			"address": address,
			"error":   err.Error(),
		})
	} else if err != redis.Nil {
		// Log cache errors but don't fail the request
		s.logger.Warn("Redis cache check failed, proceeding with API call", map[string]interface{}{
			"address": address,
			"error":   err.Error(),
		})
	}

	// CACHE MISS - Call TRM Labs API
	s.logger.Info("Cache miss, calling TRM Labs API", map[string]interface{}{
		"address": address,
		"chain":   chain,
	})

	trmResult, err := s.trmClient.ScreenAddress(ctx, address, chain)
	if err != nil {
		s.logger.Error("Failed to screen address with TRM Labs", err, map[string]interface{}{
			"address": address,
			"chain":   chain,
		})
		return nil, fmt.Errorf("failed to screen address: %w", err)
	}

	// TASK 2: CALCULATE CUSTOM RISK SCORE using Kill Switch formula
	customRiskScore := s.calculateRiskScore(trmResult)

	// Convert to our internal format with custom risk score
	result := &AMLScreeningResult{
		WalletAddress: trmResult.Address,
		Chain:         trmResult.Chain,
		RiskScore:     customRiskScore, // Use custom calculated score, NOT trmResult.RiskScore
		IsSanctioned:  trmResult.IsSanctioned,
		Flags:         trmResult.Flags,
		Details:       trmResult.Details,
		ScreenedAt:    time.Now(),
	}

	// SAVE TO REDIS CACHE (24 hour TTL)
	resultJSON, err := json.Marshal(result)
	if err != nil {
		s.logger.Warn("Failed to marshal screening result for cache", map[string]interface{}{
			"address": address,
			"error":   err.Error(),
		})
		// Don't fail the request if caching fails
	} else {
		err = s.redisClient.Set(ctx, cacheKey, resultJSON, WalletScreeningCacheTTL).Err()
		if err != nil {
			s.logger.Warn("Failed to save screening result to cache", map[string]interface{}{
				"address": address,
				"error":   err.Error(),
			})
			// Don't fail the request if caching fails
		} else {
			s.logger.Info("Screening result cached successfully", map[string]interface{}{
				"address": address,
				"ttl":     WalletScreeningCacheTTL.String(),
			})
		}
	}

	s.logger.Info("Wallet address screening completed", map[string]interface{}{
		"address":           address,
		"chain":             chain,
		"custom_risk_score": result.RiskScore,
		"trm_risk_score":    trmResult.RiskScore,
		"is_sanctioned":     result.IsSanctioned,
		"flags":             result.Flags,
	})

	return result, nil
}

// calculateRiskScore implements the "Kill Switch" scoring model
// Formula: Score = Max(Direct, Indirect) + Metadata
// - Kill Switch: IsSanctioned = immediate 100
// - Direct Risk: Based on TRM risk score and high-risk flags
// - Indirect Risk: Parsed from TRM details (mocked for now)
// - Metadata Risk: Base risk (10 points)
func (s *amlServiceImpl) calculateRiskScore(trmResult *trmlabs.ScreeningResponse) int {
	// KILL SWITCH: Sanctioned addresses get immediate 100
	if trmResult.IsSanctioned {
		s.logger.Warn("Kill switch activated: sanctioned address detected", map[string]interface{}{
			"address": trmResult.Address,
			"chain":   trmResult.Chain,
		})
		return 100
	}

	// Extract Direct Risk from TRM risk score and flags
	directRisk := s.extractDirectRisk(trmResult)

	// Extract Indirect Risk from TRM details (simplified for now)
	indirectRisk := s.extractIndirectRisk(trmResult)

	// Metadata risk (base risk for all transactions)
	metadataRisk := BaseMetadataRisk

	// Kill Switch Formula: Max(Direct, Indirect) + Metadata
	maxRisk := math.Max(float64(directRisk), float64(indirectRisk))
	finalScore := int(maxRisk) + metadataRisk

	// Cap at 100
	if finalScore > 100 {
		finalScore = 100
	}

	s.logger.Info("Custom risk score calculated", map[string]interface{}{
		"address":       trmResult.Address,
		"direct_risk":   directRisk,
		"indirect_risk": indirectRisk,
		"metadata_risk": metadataRisk,
		"final_score":   finalScore,
		"trm_score":     trmResult.RiskScore,
	})

	return finalScore
}

// extractDirectRisk calculates direct risk from TRM risk score and high-risk flags
func (s *amlServiceImpl) extractDirectRisk(trmResult *trmlabs.ScreeningResponse) int {
	directRisk := 0

	// Base direct risk from TRM score (use as-is if > 90, scale down otherwise)
	if trmResult.RiskScore > 90 {
		directRisk = trmResult.RiskScore
	} else if trmResult.RiskScore > 70 {
		directRisk = 70 + (trmResult.RiskScore-70)/2 // Scale 71-90 to 71-80
	} else {
		directRisk = trmResult.RiskScore
	}

	// Check for high-risk flags and boost score
	for _, flag := range trmResult.Flags {
		switch flag {
		case "mixer", "tornado-cash", "tumbler":
			directRisk = int(math.Max(float64(directRisk), 80))
		case "darknet", "darknet-market":
			directRisk = int(math.Max(float64(directRisk), 95))
		case "ransomware", "hack", "exploit":
			directRisk = int(math.Max(float64(directRisk), 95))
		case "sanctions", "ofac":
			// Should have been caught by IsSanctioned, but add as backup
			directRisk = 100
		case "high-risk-exchange":
			directRisk = int(math.Max(float64(directRisk), 50))
		case "gambling", "casino":
			directRisk = int(math.Max(float64(directRisk), 30))
		}
	}

	return directRisk
}

// extractIndirectRisk calculates indirect risk from TRM details
// For now, this is a simplified implementation. In production, parse details JSON
// to find indirect exposure (1-hop, 2-hop, 3-hop from risky entities)
func (s *amlServiceImpl) extractIndirectRisk(trmResult *trmlabs.ScreeningResponse) int {
	indirectRisk := 0

	// Check details for indirect exposure indicators
	if trmResult.Details != nil {
		// Example: Check if details mention indirect exposure
		if exposure, ok := trmResult.Details["indirect_exposure"]; ok {
			switch exposure {
			case "1-hop-mixer":
				indirectRisk = 40
			case "2-hop-mixer":
				indirectRisk = 20
			case "3-hop-mixer":
				indirectRisk = 10
			case "1-hop-darknet":
				indirectRisk = 50
			case "2-hop-darknet":
				indirectRisk = 25
			}
		}

		// Check for other indirect risk indicators
		if hops, ok := trmResult.Details["hops_to_risky_entity"]; ok {
			if hops == "1" {
				indirectRisk = int(math.Max(float64(indirectRisk), 40))
			} else if hops == "2" {
				indirectRisk = int(math.Max(float64(indirectRisk), 20))
			} else if hops == "3" {
				indirectRisk = int(math.Max(float64(indirectRisk), 10))
			}
		}
	}

	// If no indirect exposure found, use a minimal base
	if indirectRisk == 0 && len(trmResult.Flags) > 0 {
		// Some minimal indirect risk if any flags present
		indirectRisk = 5
	}

	return indirectRisk
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
		ID:             uuid.New().String(), // Fixed: convert UUID to string
		ActorType:      model.ActorTypeSystem,
		Action:         "aml_screening",
		ActionCategory: model.ActionCategoryPayment,
		ResourceType:   "payment",
		ResourceID:     paymentID.String(), // Fixed: convert UUID to string
		Status:         model.AuditStatusSuccess,
		Metadata:       metadata,
		CreatedAt:      time.Now(),
	}

	if err := s.auditRepo.Create(auditLog); err != nil { // Fixed: removed ctx parameter
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

	// Step 1: Evaluate velocity rules (database-backed rule engine)
	evalCtx := &RuleEvaluationContext{
		PaymentID:   paymentID,
		FromAddress: fromAddress,
		Chain:       chain,
		Amount:      amount,
		Timestamp:   time.Now(),
	}

	ruleResults, err := s.ruleEngine.EvaluateVelocityRules(ctx, evalCtx)
	if err != nil {
		s.logger.Error("Failed to evaluate velocity rules", err, map[string]interface{}{
			"payment_id":   paymentID,
			"from_address": fromAddress,
		})
	} else {
		shouldBlock, blockingRule := s.ruleEngine.ShouldBlockTransaction(ruleResults)
		if shouldBlock {
			s.logger.Warn("Transaction blocked by velocity rule", map[string]interface{}{
				"payment_id":   paymentID,
				"from_address": fromAddress,
				"rule_id":      blockingRule.RuleID,
				"rule_name":    blockingRule.RuleName,
				"reason":       blockingRule.Reason,
			})
			return ErrVelocityLimitExceeded
		}
	}

	// Step 2: Screen the wallet address
	result, err := s.ScreenWalletAddress(ctx, fromAddress, chain)
	if err != nil {
		return fmt.Errorf("AML screening failed: %w", err)
	}

	// Step 3: Record the screening result
	if err := s.RecordScreeningResult(ctx, paymentID, result); err != nil {
		s.logger.Error("Failed to record screening result", err, nil)
		// Don't fail the transaction if we can't record, just log
	}

	// Step 4: Check if address is sanctioned (Kill Switch)
	if result.IsSanctioned && s.autoRejectSanctioned {
		s.logger.Warn("Transaction BLOCKED: sanctioned address (Kill Switch)", map[string]interface{}{
			"payment_id":   paymentID,
			"from_address": fromAddress,
			"risk_score":   result.RiskScore,
			"flags":        result.Flags,
		})
		return ErrSanctionedAddress
	}

	// TASK 3: Check risk score with NEW THRESHOLDS
	// >= 86: Block (Critical Risk)
	// 61-85: Hold for Manual Review (High Risk)
	// <= 60: Allow (with warning if > 20)

	if result.RiskScore >= RiskScoreCriticalThreshold { // >= 86
		s.logger.Error("Transaction BLOCKED: critical risk score", map[string]interface{}{
			"payment_id":   paymentID,
			"from_address": fromAddress,
			"risk_score":   result.RiskScore,
			"threshold":    RiskScoreCriticalThreshold,
			"flags":        result.Flags,
		})
		return ErrRiskCritical
	}

	if result.RiskScore >= RiskScoreHighThreshold { // 61-85
		s.logger.Warn("Transaction ON HOLD: high risk score, manual review required", map[string]interface{}{
			"payment_id":   paymentID,
			"from_address": fromAddress,
			"risk_score":   result.RiskScore,
			"threshold":    RiskScoreHighThreshold,
			"flags":        result.Flags,
			"action":       "hold_for_review",
		})
		return ErrRiskHigh
	}

	// Risk score < 61: Allow, but log warning if > 20
	if result.RiskScore > RiskScoreMediumThreshold { // 21-60
		s.logger.Warn("Transaction ALLOWED but elevated risk detected", map[string]interface{}{
			"payment_id":   paymentID,
			"from_address": fromAddress,
			"risk_score":   result.RiskScore,
			"flags":        result.Flags,
			"action":       "allow_with_warning",
		})
	}

	// Step 6: Check for specific high-risk flags
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

	s.logger.Info("Transaction PASSED AML validation", map[string]interface{}{
		"payment_id":      paymentID,
		"from_address":    fromAddress,
		"risk_score":      result.RiskScore,
		"velocity_check":  "passed",
		"sanctions_check": "passed",
		"decision":        "allow",
	})

	return nil
}

// GetScreeningHistory retrieves the screening history for a wallet address
func (s *amlServiceImpl) GetScreeningHistory(ctx context.Context, walletAddress string) ([]*AMLScreeningResult, error) {
	// Query audit logs for AML screening records
	// This is a simplified implementation - in production, you might want a dedicated table
	action := "aml_screening"
	resourceType := "payment"
	auditLogs, err := s.auditRepo.List(repository.AuditFilter{ // Fixed: removed ctx parameter
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
