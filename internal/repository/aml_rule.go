package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
)

type AMLRuleRepository struct {
	db     *gorm.DB
	logger *logger.Logger
}

func NewAMLRuleRepository(db *gorm.DB, logger *logger.Logger) *AMLRuleRepository {
	return &AMLRuleRepository{
		db:     db,
		logger: logger,
	}
}

func (r *AMLRuleRepository) GetByID(ctx context.Context, id string) (*model.AMLRule, error) {
	var rule model.AMLRule
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error; err != nil {
		return nil, fmt.Errorf("failed to get AML rule: %w", err)
	}
	return &rule, nil
}

func (r *AMLRuleRepository) GetEnabledByCategory(ctx context.Context, category model.AMLRuleCategory) ([]*model.AMLRule, error) {
	var rules []*model.AMLRule
	if err := r.db.WithContext(ctx).
		Where("category = ? AND enabled = ?", category, true).
		Order("severity DESC, created_at ASC").
		Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("failed to get enabled AML rules: %w", err)
	}
	return rules, nil
}

func (r *AMLRuleRepository) GetAll(ctx context.Context) ([]*model.AMLRule, error) {
	var rules []*model.AMLRule
	if err := r.db.WithContext(ctx).
		Order("category, severity DESC, created_at ASC").
		Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("failed to get all AML rules: %w", err)
	}
	return rules, nil
}

func (r *AMLRuleRepository) Create(ctx context.Context, rule *model.AMLRule) error {
	if err := r.db.WithContext(ctx).Create(rule).Error; err != nil {
		return fmt.Errorf("failed to create AML rule: %w", err)
	}
	return nil
}

func (r *AMLRuleRepository) Update(ctx context.Context, rule *model.AMLRule) error {
	if err := r.db.WithContext(ctx).Save(rule).Error; err != nil {
		return fmt.Errorf("failed to update AML rule: %w", err)
	}
	return nil
}

func (r *AMLRuleRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&model.AMLRule{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete AML rule: %w", err)
	}
	return nil
}

func (r *AMLRuleRepository) UpdateEnabled(ctx context.Context, id string, enabled bool) error {
	if err := r.db.WithContext(ctx).
		Model(&model.AMLRule{}).
		Where("id = ?", id).
		Update("enabled", enabled).Error; err != nil {
		return fmt.Errorf("failed to update rule enabled status: %w", err)
	}
	return nil
}

func (r *AMLRuleRepository) IncrementTriggerCount(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE aml_rules 
		SET effectiveness_stats = jsonb_set(
			COALESCE(effectiveness_stats, '{}'::jsonb),
			'{total_triggers}',
			(COALESCE((effectiveness_stats->>'total_triggers')::int, 0) + 1)::text::jsonb
		)
		WHERE id = ?
	`, id).Error
}
