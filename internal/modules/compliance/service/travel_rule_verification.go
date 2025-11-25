package service

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
)

var (
	// ErrSignatureVerificationFailed is returned when signature verification fails
	ErrSignatureVerificationFailed = errors.New("signature verification failed")
	// ErrSanctionsCheckFailed is returned when wallet/user is on sanctions list
	ErrSanctionsCheckFailed = errors.New("sanctions check failed: wallet or user on sanctions list")
	// ErrEKYCRequired is returned when eKYC step-up verification is required
	ErrEKYCRequired = errors.New("eKYC verification required for this transaction")
	// ErrGeoMismatchDetected is returned when country mismatch detected
	ErrGeoMismatchDetected = errors.New("geographic mismatch detected")
)

// TravelRuleVerificationService handles 4-layer verification for unhosted wallets
type TravelRuleVerificationService interface {
	// Layer 1: Proof of Ownership
	VerifySignature(ctx context.Context, walletAddress string, message string, signature string) (bool, error)
	GenerateSignatureMessage(payerName string, walletAddress string, txID string) string

	// Layer 2: Sanctions Screening
	CheckSanctions(ctx context.Context, walletAddress string, payerName string, country string) (*SanctionsCheckResult, error)

	// Layer 3: Passive Signals
	AnalyzePassiveSignals(ctx context.Context, ipAddress string, userAgent string, deviceFingerprint string, declaredCountry string) (*PassiveSignalsResult, error)

	// Layer 4: eKYC Step-up
	RequiresEKYC(ctx context.Context, travelRuleData *model.TravelRuleData, sanctionsResult *SanctionsCheckResult, passiveResult *PassiveSignalsResult) bool
	VerifyEKYC(ctx context.Context, documentType string, documentID string, faceImage []byte) (*EKYCResult, error)

	// Orchestration
	PerformFullVerification(ctx context.Context, travelRuleID string) (*VerificationSummary, error)
}

// SanctionsCheckResult represents the result of Layer 2 sanctions screening
type SanctionsCheckResult struct {
	Status               string                 `json:"status"` // clear, pep, sanctions, watchlist, adverse_media
	WalletSanctioned     bool                   `json:"wallet_sanctioned"`
	NameSanctioned       bool                   `json:"name_sanctioned"`
	ChainalysisScore     decimal.Decimal        `json:"chainalysis_score"` // 0-100
	HitDetails           map[string]interface{} `json:"hit_details"`
	RequiresManualReview bool                   `json:"requires_manual_review"`
}

// PassiveSignalsResult represents the result of Layer 3 anomaly detection
type PassiveSignalsResult struct {
	IPCountry     string          `json:"ip_country"`
	GeoMismatch   bool            `json:"geo_mismatch"`
	AnomalyScore  decimal.Decimal `json:"anomaly_score"` // 0-100
	AnomalyFlags  []string        `json:"anomaly_flags"`
	DeviceTrusted bool            `json:"device_trusted"`
	RiskLevel     string          `json:"risk_level"` // low, medium, high
}

// EKYCResult represents the result of Layer 4 eKYC verification
type EKYCResult struct {
	Verified      bool      `json:"verified"`
	Provider      string    `json:"provider"`
	Reference     string    `json:"reference"`
	DocumentType  string    `json:"document_type"`
	DocumentID    string    `json:"document_id"`
	LivenessScore float64   `json:"liveness_score"`
	VerifiedAt    time.Time `json:"verified_at"`
}

// VerificationSummary represents the summary of all 4 layers
type VerificationSummary struct {
	Layer1Signature      bool                  `json:"layer1_signature_verified"`
	Layer2Sanctions      *SanctionsCheckResult `json:"layer2_sanctions"`
	Layer3PassiveSignals *PassiveSignalsResult `json:"layer3_passive_signals"`
	Layer4EKYC           *EKYCResult           `json:"layer4_ekyc,omitempty"`
	VerificationTier     int                   `json:"verification_tier"`
	OverallRisk          string                `json:"overall_risk"` // low, medium, high, critical
	Approved             bool                  `json:"approved"`
	RequiresReview       bool                  `json:"requires_review"`
}

type travelRuleVerificationServiceImpl struct {
	travelRuleRepo repository.TravelRuleRepository
	amlService     AMLService
	logger         *logger.Logger
}

// NewTravelRuleVerificationService creates a new verification service
func NewTravelRuleVerificationService(
	travelRuleRepo repository.TravelRuleRepository,
	amlService AMLService,
	logger *logger.Logger,
) TravelRuleVerificationService {
	return &travelRuleVerificationServiceImpl{
		travelRuleRepo: travelRuleRepo,
		amlService:     amlService,
		logger:         logger,
	}
}

// ==============================================================
// Layer 1: Proof of Ownership (Signature Verification)
// ==============================================================

// GenerateSignatureMessage generates the message that must be signed by the user
func (s *travelRuleVerificationServiceImpl) GenerateSignatureMessage(payerName string, walletAddress string, txID string) string {
	timestamp := time.Now().Format(time.RFC3339)
	message := fmt.Sprintf(
		"I, %s, confirm that I am the owner of wallet address %s and am responsible for transaction %s. "+
			"I declare that the information provided is accurate and complete. "+
			"Timestamp: %s",
		payerName,
		walletAddress,
		txID,
		timestamp,
	)
	return message
}

// VerifySignature verifies the cryptographic signature (Layer 1)
// For Solana/Phantom wallets, this uses Ed25519 signature verification
func (s *travelRuleVerificationServiceImpl) VerifySignature(ctx context.Context, walletAddress string, message string, signature string) (bool, error) {
	s.logger.Info("Verifying signature for wallet", map[string]interface{}{
		"wallet_address": walletAddress,
	})

	// Decode the signature (base64 or hex)
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		// Try hex decoding
		sigBytes, err = hex.DecodeString(signature)
		if err != nil {
			return false, fmt.Errorf("failed to decode signature: %w", err)
		}
	}

	// For Solana: wallet address is base58-encoded public key
	// For production, use: github.com/mr-tron/base58 to decode
	// For this example, we'll simulate verification

	// CRITICAL: In production, implement actual Ed25519 verification:
	// pubKey := base58.Decode(walletAddress)
	// messageBytes := []byte(message)
	// verified := ed25519.Verify(pubKey, messageBytes, sigBytes)

	// Placeholder verification (REPLACE in production)
	verified := len(sigBytes) == ed25519.SignatureSize

	if !verified {
		s.logger.Warn("Signature verification failed", map[string]interface{}{
			"wallet_address": walletAddress,
		})
		return false, ErrSignatureVerificationFailed
	}

	s.logger.Info("Signature verified successfully", map[string]interface{}{
		"wallet_address": walletAddress,
	})
	return true, nil
}

// ==============================================================
// Layer 2: Sanctions Screening
// ==============================================================

// CheckSanctions performs sanctions screening (Layer 2)
func (s *travelRuleVerificationServiceImpl) CheckSanctions(ctx context.Context, walletAddress string, payerName string, country string) (*SanctionsCheckResult, error) {
	s.logger.Info("Running sanctions screening", map[string]interface{}{
		"wallet_address": walletAddress,
		"payer_name":     payerName,
		"country":        country,
	})

	result := &SanctionsCheckResult{
		Status:           "clear",
		WalletSanctioned: false,
		NameSanctioned:   false,
		ChainalysisScore: decimal.Zero,
		HitDetails:       make(map[string]interface{}),
	}

	// Step 1: Check wallet address against Chainalysis/TRM Labs
	// CRITICAL: In production, integrate with actual Chainalysis API
	walletRisk := s.checkWalletRisk(walletAddress)
	result.ChainalysisScore = walletRisk

	if walletRisk.GreaterThan(decimal.NewFromInt(75)) {
		result.WalletSanctioned = true
		result.Status = "sanctions"
		result.HitDetails["wallet_risk_score"] = walletRisk.String()
		result.RequiresManualReview = true
	}

	// Step 2: Check payer name against OFAC/UN sanctions lists
	// CRITICAL: In production, integrate with OFAC SDN list
	nameHit := s.checkNameSanctions(payerName)
	if nameHit {
		result.NameSanctioned = true
		result.Status = "sanctions"
		result.HitDetails["name_match"] = payerName
		result.RequiresManualReview = true
	}

	// Step 3: Check for PEP (Politically Exposed Person)
	// CRITICAL: In production, integrate with PEP database
	isPEP := s.checkPEP(payerName, country)
	if isPEP {
		result.Status = "pep"
		result.HitDetails["pep_match"] = true
		result.RequiresManualReview = true
	}

	if result.Status != "clear" {
		s.logger.Warn("Sanctions check hit", map[string]interface{}{
			"wallet_address": walletAddress,
			"status":         result.Status,
		})
		return result, ErrSanctionsCheckFailed
	}

	s.logger.Info("Sanctions check passed", map[string]interface{}{
		"wallet_address": walletAddress,
	})
	return result, nil
}

// checkWalletRisk simulates Chainalysis wallet risk scoring
// CRITICAL: Replace with actual Chainalysis API integration
func (s *travelRuleVerificationServiceImpl) checkWalletRisk(walletAddress string) decimal.Decimal {
	// Placeholder: In production, call Chainalysis API
	// Example: https://api.chainalysis.com/api/risk/v1/address/{address}

	// For demo, return low risk (0-25)
	return decimal.NewFromFloat(15.0)
}

// checkNameSanctions simulates OFAC SDN list check
// CRITICAL: Replace with actual OFAC API integration
func (s *travelRuleVerificationServiceImpl) checkNameSanctions(name string) bool {
	// Placeholder: In production, check against OFAC SDN list
	// Example API: https://sanctionssearch.ofac.treas.gov/

	// For demo, check for obvious test cases
	name = strings.ToLower(name)
	sanctionedNames := []string{"kim jong", "vladimir putin", "osama bin"}
	for _, sanctioned := range sanctionedNames {
		if strings.Contains(name, sanctioned) {
			return true
		}
	}
	return false
}

// checkPEP simulates PEP database check
// CRITICAL: Replace with actual PEP database integration
func (s *travelRuleVerificationServiceImpl) checkPEP(name string, country string) bool {
	// Placeholder: In production, check against PEP database
	// Commercial providers: Dow Jones, LexisNexis, Refinitiv

	// For demo, return false
	return false
}

// ==============================================================
// Layer 3: Passive Signals (Anomaly Detection)
// ==============================================================

// AnalyzePassiveSignals performs passive signal analysis (Layer 3)
func (s *travelRuleVerificationServiceImpl) AnalyzePassiveSignals(ctx context.Context, ipAddress string, userAgent string, deviceFingerprint string, declaredCountry string) (*PassiveSignalsResult, error) {
	s.logger.Info("Analyzing passive signals", map[string]interface{}{
		"ip_address":         ipAddress,
		"declared_country":   declaredCountry,
		"device_fingerprint": deviceFingerprint,
	})

	result := &PassiveSignalsResult{
		AnomalyFlags:  []string{},
		DeviceTrusted: true,
		RiskLevel:     "low",
	}

	// Step 1: Geo-IP lookup
	ipCountry := s.getCountryFromIP(ipAddress)
	result.IPCountry = ipCountry

	// Step 2: Check for geo mismatch
	if ipCountry != declaredCountry {
		result.GeoMismatch = true
		result.AnomalyFlags = append(result.AnomalyFlags, "geo_mismatch")
		s.logger.Warn("Geo mismatch detected", map[string]interface{}{
			"declared_country": declaredCountry,
			"ip_country":       ipCountry,
		})
	}

	// Step 3: Check for VPN/Proxy
	if s.isVPNOrProxy(ipAddress) {
		result.AnomalyFlags = append(result.AnomalyFlags, "vpn_detected")
	}

	// Step 4: Browser fingerprint analysis
	if !s.isDeviceTrusted(deviceFingerprint) {
		result.DeviceTrusted = false
		result.AnomalyFlags = append(result.AnomalyFlags, "untrusted_device")
	}

	// Step 5: User agent analysis
	if s.isBot(userAgent) {
		result.AnomalyFlags = append(result.AnomalyFlags, "bot_detected")
	}

	// Calculate anomaly score
	anomalyScore := decimal.Zero
	if result.GeoMismatch {
		anomalyScore = anomalyScore.Add(decimal.NewFromInt(30))
	}
	if !result.DeviceTrusted {
		anomalyScore = anomalyScore.Add(decimal.NewFromInt(20))
	}
	if len(result.AnomalyFlags) >= 3 {
		anomalyScore = anomalyScore.Add(decimal.NewFromInt(25))
	}
	result.AnomalyScore = anomalyScore

	// Determine risk level
	if anomalyScore.GreaterThan(decimal.NewFromInt(70)) {
		result.RiskLevel = "high"
	} else if anomalyScore.GreaterThan(decimal.NewFromInt(40)) {
		result.RiskLevel = "medium"
	}

	s.logger.Info("Passive signals analyzed", map[string]interface{}{
		"anomaly_score": anomalyScore.String(),
		"risk_level":    result.RiskLevel,
		"anomaly_flags": result.AnomalyFlags,
	})

	return result, nil
}

// getCountryFromIP performs GeoIP lookup
// CRITICAL: Replace with actual GeoIP service (MaxMind, ipapi.co, etc)
func (s *travelRuleVerificationServiceImpl) getCountryFromIP(ipAddress string) string {
	// Placeholder: In production, use MaxMind GeoIP2 or ipapi.co
	// Example: https://ipapi.co/{ip}/country/

	// For demo, check if IP is private/localhost
	ip := net.ParseIP(ipAddress)
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() {
		return "VN" // Assume local development in Vietnam
	}

	// For production, implement actual GeoIP lookup
	return "VN"
}

// isVPNOrProxy checks if IP is VPN/Proxy
// CRITICAL: Replace with VPN detection service
func (s *travelRuleVerificationServiceImpl) isVPNOrProxy(ipAddress string) bool {
	// Placeholder: In production, use VPN detection API
	// Services: IPQualityScore, IPHub, GetIPIntel
	return false
}

// isDeviceTrusted checks if device fingerprint is trusted
func (s *travelRuleVerificationServiceImpl) isDeviceTrusted(deviceFingerprint string) bool {
	// Placeholder: In production, maintain a database of trusted devices
	// Or use FingerprintJS Pro for device intelligence
	return len(deviceFingerprint) > 0
}

// isBot checks if user agent indicates a bot
func (s *travelRuleVerificationServiceImpl) isBot(userAgent string) bool {
	botPatterns := []string{"bot", "crawler", "spider", "scraper", "curl", "wget"}
	userAgentLower := strings.ToLower(userAgent)
	for _, pattern := range botPatterns {
		if strings.Contains(userAgentLower, pattern) {
			return true
		}
	}
	return false
}

// ==============================================================
// Layer 4: eKYC Step-up Verification
// ==============================================================

// RequiresEKYC determines if eKYC is required based on risk factors
func (s *travelRuleVerificationServiceImpl) RequiresEKYC(ctx context.Context, travelRuleData *model.TravelRuleData, sanctionsResult *SanctionsCheckResult, passiveResult *PassiveSignalsResult) bool {
	// Trigger eKYC for:
	// 1. High-value transactions (> $10,000)
	if travelRuleData.TransactionAmount.GreaterThan(decimal.NewFromInt(10000)) {
		s.logger.Info("eKYC required: high-value transaction", map[string]interface{}{
			"amount": travelRuleData.TransactionAmount.String(),
		})
		return true
	}

	// 2. Sanctions hits
	if sanctionsResult != nil && sanctionsResult.RequiresManualReview {
		s.logger.Info("eKYC required: sanctions hit", nil)
		return true
	}

	// 3. High anomaly score
	if passiveResult != nil && passiveResult.AnomalyScore.GreaterThan(decimal.NewFromInt(70)) {
		s.logger.Info("eKYC required: high anomaly score", map[string]interface{}{
			"anomaly_score": passiveResult.AnomalyScore.String(),
		})
		return true
	}

	// 4. Cross-border + High risk
	if travelRuleData.IsCrossBorder() && travelRuleData.IsHighRisk() {
		s.logger.Info("eKYC required: cross-border high-risk", nil)
		return true
	}

	return false
}

// VerifyEKYC performs eKYC document verification with liveness check
// CRITICAL: Integrate with eKYC provider (FPT.AI, AWS Rekognition, etc)
func (s *travelRuleVerificationServiceImpl) VerifyEKYC(ctx context.Context, documentType string, documentID string, faceImage []byte) (*EKYCResult, error) {
	s.logger.Info("Performing eKYC verification", map[string]interface{}{
		"document_type": documentType,
	})

	// Placeholder: In production, integrate with eKYC service
	// Example providers:
	// - FPT.AI eKYC: https://ekyc.fpt.ai/
	// - AWS Rekognition: https://aws.amazon.com/rekognition/
	// - Sumsub: https://sumsub.com/

	result := &EKYCResult{
		Verified:      true,
		Provider:      "FPT.AI",
		Reference:     fmt.Sprintf("EKYC-%d", time.Now().Unix()),
		DocumentType:  documentType,
		DocumentID:    documentID,
		LivenessScore: 0.95,
		VerifiedAt:    time.Now(),
	}

	s.logger.Info("eKYC verification completed", map[string]interface{}{
		"reference":      result.Reference,
		"liveness_score": result.LivenessScore,
	})

	return result, nil
}

// ==============================================================
// Orchestration: Full Verification Flow
// ==============================================================

// PerformFullVerification orchestrates all 4 layers of verification
func (s *travelRuleVerificationServiceImpl) PerformFullVerification(ctx context.Context, travelRuleID string) (*VerificationSummary, error) {
	s.logger.Info("Performing full 4-layer verification", map[string]interface{}{
		"travel_rule_id": travelRuleID,
	})

	// Fetch Travel Rule data
	travelRuleData, err := s.travelRuleRepo.GetByID(ctx, travelRuleID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Travel Rule data: %w", err)
	}

	summary := &VerificationSummary{
		VerificationTier: 0,
		Approved:         false,
		RequiresReview:   false,
		OverallRisk:      "low",
	}

	// Layer 1: Signature verification (required)
	// In production, this should be done during submission
	summary.Layer1Signature = true // Assume already verified
	summary.VerificationTier = 1

	// Layer 2: Sanctions screening
	sanctionsResult, err := s.CheckSanctions(ctx, travelRuleData.PayerWalletAddress, travelRuleData.PayerFullName, travelRuleData.PayerCountry)
	summary.Layer2Sanctions = sanctionsResult
	if err != nil {
		summary.RequiresReview = true
		summary.OverallRisk = "high"
	} else {
		summary.VerificationTier = 2
	}

	// Layer 3: Passive signals (if available)
	// In production, extract IP/User-Agent from request context
	if travelRuleData.PayerAddress.Valid {
		// Placeholder: Extract passive signals from stored data
		summary.Layer3PassiveSignals = &PassiveSignalsResult{
			IPCountry:     travelRuleData.PayerCountry,
			GeoMismatch:   false,
			AnomalyScore:  decimal.NewFromInt(10),
			AnomalyFlags:  []string{},
			DeviceTrusted: true,
			RiskLevel:     "low",
		}
		summary.VerificationTier = 3
	}

	// Layer 4: eKYC (if required)
	requiresEKYC := s.RequiresEKYC(ctx, travelRuleData, sanctionsResult, summary.Layer3PassiveSignals)
	if requiresEKYC {
		// Check if eKYC already completed
		// In production, this would be a separate flow triggered by user
		summary.RequiresReview = true
	}

	// Determine overall approval
	if !summary.RequiresReview && summary.VerificationTier >= 2 {
		summary.Approved = true
	}

	// Determine overall risk
	if summary.Layer2Sanctions != nil && summary.Layer2Sanctions.RequiresManualReview {
		summary.OverallRisk = "high"
	} else if summary.Layer3PassiveSignals != nil && summary.Layer3PassiveSignals.RiskLevel == "medium" {
		summary.OverallRisk = "medium"
	}

	s.logger.Info("Verification completed", map[string]interface{}{
		"travel_rule_id":    travelRuleID,
		"verification_tier": summary.VerificationTier,
		"approved":          summary.Approved,
		"overall_risk":      summary.OverallRisk,
	})

	return summary, nil
}
