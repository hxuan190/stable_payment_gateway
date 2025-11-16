package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

// RiskAssessmentService handles risk scoring and assessment
type RiskAssessmentService interface {
	AssessRisk(ctx context.Context, submission *model.KYCSubmission, verificationResults []*model.KYCVerificationResult) (*model.KYCRiskAssessment, error)
}

type riskAssessmentService struct {
}

// NewRiskAssessmentService creates a new risk assessment service
func NewRiskAssessmentService() RiskAssessmentService {
	return &riskAssessmentService{}
}

func (s *riskAssessmentService) AssessRisk(ctx context.Context, submission *model.KYCSubmission, verificationResults []*model.KYCVerificationResult) (*model.KYCRiskAssessment, error) {
	assessment := &model.KYCRiskAssessment{
		KYCSubmissionID: submission.ID,
		MerchantID:      submission.MerchantID,
		RiskLevel:       model.RiskLevelLow,
		RiskScore:       0,
		AssessedAt:      time.Now(),
	}

	// Calculate risk score based on various factors
	riskScore := 0
	riskFactors := model.RiskFactorsMap{}

	// 1. Check verification results
	for _, vr := range verificationResults {
		if vr.Passed != nil && !*vr.Passed {
			riskScore += 20
			switch vr.VerificationType {
			case model.VerificationTypeFaceMatch:
				riskFactors.DocumentQualityIssues = true
			case model.VerificationTypeAgeCheck:
				riskFactors.AgeBelowThreshold = true
			case model.VerificationTypeNameMatch:
				riskFactors.NameMismatch = true
			case model.VerificationTypeSanctionsCheck:
				riskFactors.SanctionedEntity = true
				riskScore += 50 // Sanctions hit is critical
			}
		}
	}

	// 2. Check industry risk
	if submission.ProductCategory != nil {
		category := *submission.ProductCategory
		if isHighRiskIndustry(category) {
			riskScore += 30
			riskFactors.HighRiskIndustry = true
			assessment.IsHighRiskIndustry = true
			assessment.IndustryCategory = submission.ProductCategory
		}
	}

	// 3. Check business patterns for companies
	if submission.MerchantType == model.MerchantTypeCompany {
		// Companies get slightly higher base risk
		riskScore += 5
	}

	// 4. Sanctions check result
	sanctionsResult := findVerificationResult(verificationResults, model.VerificationTypeSanctionsCheck)
	if sanctionsResult != nil && sanctionsResult.Passed != nil && !*sanctionsResult.Passed {
		assessment.SanctionsChecked = true
		assessment.SanctionsHit = true
		riskFactors.SanctionedEntity = true

		// Store sanctions details
		assessment.SanctionsDetails = sanctionsResult.ResultData
	} else {
		assessment.SanctionsChecked = true
		assessment.SanctionsHit = false
	}

	// Determine risk level based on score
	if riskScore >= 70 {
		assessment.RiskLevel = model.RiskLevelCritical
	} else if riskScore >= 40 {
		assessment.RiskLevel = model.RiskLevelHigh
	} else if riskScore >= 20 {
		assessment.RiskLevel = model.RiskLevelMedium
	} else {
		assessment.RiskLevel = model.RiskLevelLow
	}

	assessment.RiskScore = riskScore

	// Store risk factors as JSON
	riskFactorsJSON, _ := json.Marshal(riskFactors)
	riskFactorsStr := string(riskFactorsJSON)
	assessment.RiskFactors = &riskFactorsStr

	// Determine recommended action
	if assessment.RiskLevel == model.RiskLevelCritical || assessment.SanctionsHit {
		assessment.RecommendedAction = model.ActionReject
	} else if assessment.RiskLevel == model.RiskLevelHigh || assessment.RiskLevel == model.RiskLevelMedium {
		assessment.RecommendedAction = model.ActionManualReview
	} else {
		assessment.RecommendedAction = model.ActionAutoApprove
	}

	// Set recommended limits based on merchant type and risk level
	limits := calculateRecommendedLimits(submission.MerchantType, assessment.RiskLevel)
	limitsJSON, _ := json.Marshal(limits)
	limitsStr := string(limitsJSON)
	assessment.RecommendedLimits = &limitsStr

	return assessment, nil
}

// ===== Helper Functions =====

func isHighRiskIndustry(category string) bool {
	highRiskCategories := map[string]bool{
		"gambling":     true,
		"casino":       true,
		"crypto":       true,
		"otc":          true,
		"adult":        true,
		"forex":        true,
		"loan":         true,
		"pawn_shop":    true,
		"money_exchange": true,
	}

	return highRiskCategories[category]
}

func findVerificationResult(results []*model.KYCVerificationResult, verificationType string) *model.KYCVerificationResult {
	for _, r := range results {
		if r.VerificationType == verificationType {
			return r
		}
	}
	return nil
}

func calculateRecommendedLimits(merchantType model.MerchantType, riskLevel model.RiskLevel) model.RecommendedLimitsMap {
	var baseLimits model.RecommendedLimitsMap

	// Base limits by merchant type
	switch merchantType {
	case model.MerchantTypeIndividual:
		baseLimits = model.RecommendedLimitsMap{
			DailyLimitVND:   200000000,  // 200M VND
			MonthlyLimitVND: 3000000000, // 3B VND
		}
	case model.MerchantTypeHouseholdBusiness:
		baseLimits = model.RecommendedLimitsMap{
			DailyLimitVND:   500000000,   // 500M VND
			MonthlyLimitVND: 10000000000, // 10B VND
		}
	case model.MerchantTypeCompany:
		baseLimits = model.RecommendedLimitsMap{
			DailyLimitVND:   1000000000,  // 1B VND
			MonthlyLimitVND: 30000000000, // 30B VND
		}
	default:
		baseLimits = model.RecommendedLimitsMap{
			DailyLimitVND:   200000000,
			MonthlyLimitVND: 3000000000,
		}
	}

	// Adjust based on risk level
	switch riskLevel {
	case model.RiskLevelLow:
		// Keep base limits
		return baseLimits
	case model.RiskLevelMedium:
		// Reduce limits by 30%
		return model.RecommendedLimitsMap{
			DailyLimitVND:   baseLimits.DailyLimitVND * 0.7,
			MonthlyLimitVND: baseLimits.MonthlyLimitVND * 0.7,
		}
	case model.RiskLevelHigh:
		// Reduce limits by 50%
		return model.RecommendedLimitsMap{
			DailyLimitVND:   baseLimits.DailyLimitVND * 0.5,
			MonthlyLimitVND: baseLimits.MonthlyLimitVND * 0.5,
		}
	case model.RiskLevelCritical:
		// Minimal limits
		return model.RecommendedLimitsMap{
			DailyLimitVND:   50000000,   // 50M VND
			MonthlyLimitVND: 500000000,  // 500M VND
		}
	default:
		return baseLimits
	}
}
