package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	complianceRepository "github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/repository"
	paymentdomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
)

type RuleEngine struct {
	ruleRepo    *complianceRepository.AMLRuleRepository
	paymentRepo paymentdomain.PaymentRepository
	logger      *logger.Logger
}

func NewRuleEngine(
	ruleRepo *complianceRepository.AMLRuleRepository,
	paymentRepo paymentdomain.PaymentRepository,
	logger *logger.Logger,
) *RuleEngine {
	return &RuleEngine{
		ruleRepo:    ruleRepo,
		paymentRepo: paymentRepo,
		logger:      logger,
	}
}

type RuleEvaluationContext struct {
	PaymentID   uuid.UUID
	FromAddress string
	Chain       string
	Amount      decimal.Decimal
	Timestamp   time.Time
}

type RuleEvaluationResult struct {
	RuleID    string
	RuleName  string
	Triggered bool
	Severity  model.AMLRuleSeverity
	Actions   []model.RuleAction
	Reason    string
}

func (e *RuleEngine) EvaluateVelocityRules(ctx context.Context, evalCtx *RuleEvaluationContext) ([]*RuleEvaluationResult, error) {
	rules, err := e.ruleRepo.GetEnabledByCategory(ctx, model.AMLRuleCategoryVelocity)
	if err != nil {
		return nil, fmt.Errorf("failed to get velocity rules: %w", err)
	}

	var results []*RuleEvaluationResult

	for _, rule := range rules {
		result, err := e.evaluateVelocityRule(ctx, rule, evalCtx)
		if err != nil {
			e.logger.Error("Failed to evaluate velocity rule", err, map[string]interface{}{
				"rule_id":      rule.ID,
				"from_address": evalCtx.FromAddress,
			})
			continue
		}

		if result != nil {
			results = append(results, result)
			if result.Triggered {
				if err := e.ruleRepo.IncrementTriggerCount(ctx, rule.ID); err != nil {
					e.logger.Error("Failed to increment trigger count", err, map[string]interface{}{
						"rule_id": rule.ID,
					})
				}
			}
		}
	}

	return results, nil
}

func (e *RuleEngine) evaluateVelocityRule(ctx context.Context, rule *model.AMLRule, evalCtx *RuleEvaluationContext) (*RuleEvaluationResult, error) {
	var conditions model.VelocityRuleConditions
	if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
		return nil, fmt.Errorf("failed to parse velocity rule conditions: %w", err)
	}

	var actions []model.RuleAction
	if err := json.Unmarshal(rule.Actions, &actions); err != nil {
		return nil, fmt.Errorf("failed to parse rule actions: %w", err)
	}

	result := &RuleEvaluationResult{
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Triggered: false,
		Severity:  rule.Severity,
		Actions:   actions,
	}

	switch conditions.Field {
	case "tx_count_24h":
		txCount, err := e.paymentRepo.CountByAddressAndWindow(
			evalCtx.FromAddress,
			time.Duration(conditions.WindowHours)*time.Hour,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to count transactions: %w", err)
		}

		if txCount >= int64(conditions.MaxTransactions) {
			result.Triggered = true
			result.Reason = fmt.Sprintf("Transaction count (%d) exceeds limit (%d) in %d hours",
				txCount, conditions.MaxTransactions, conditions.WindowHours)
		}

	case "tx_count_1h":
		txCount, err := e.paymentRepo.CountByAddressAndWindow(
			evalCtx.FromAddress,
			time.Duration(conditions.WindowHours)*time.Hour,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to count transactions: %w", err)
		}

		if txCount >= int64(conditions.MaxTransactions) {
			result.Triggered = true
			result.Reason = fmt.Sprintf("High-frequency pattern detected: %d transactions in %d hour(s)",
				txCount, conditions.WindowHours)
		}

	default:
		e.logger.Warn("Unknown velocity rule field", map[string]interface{}{
			"rule_id": rule.ID,
			"field":   conditions.Field,
		})
		return nil, nil
	}

	return result, nil
}

func (e *RuleEngine) ShouldBlockTransaction(results []*RuleEvaluationResult) (bool, *RuleEvaluationResult) {
	for _, result := range results {
		if !result.Triggered {
			continue
		}

		for _, action := range result.Actions {
			if action.Type == "block_transaction" {
				return true, result
			}
		}
	}
	return false, nil
}

func (e *RuleEngine) ShouldRequireReview(results []*RuleEvaluationResult) (bool, *RuleEvaluationResult) {
	for _, result := range results {
		if !result.Triggered {
			continue
		}

		for _, action := range result.Actions {
			if action.Type == "require_review" {
				return true, result
			}
		}
	}
	return false, nil
}
