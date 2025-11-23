package model

import (
	"encoding/json"
	"time"
)

type AMLRuleCategory string

const (
	AMLRuleCategoryThreshold  AMLRuleCategory = "threshold"
	AMLRuleCategoryPattern    AMLRuleCategory = "pattern"
	AMLRuleCategoryVelocity   AMLRuleCategory = "velocity"
	AMLRuleCategoryBehavioral AMLRuleCategory = "behavioral"
	AMLRuleCategorySanctions  AMLRuleCategory = "sanctions"
	AMLRuleCategoryWallet     AMLRuleCategory = "wallet"
)

type AMLRuleSeverity string

const (
	AMLRuleSeverityLow      AMLRuleSeverity = "LOW"
	AMLRuleSeverityMedium   AMLRuleSeverity = "MEDIUM"
	AMLRuleSeverityHigh     AMLRuleSeverity = "HIGH"
	AMLRuleSeverityCritical AMLRuleSeverity = "CRITICAL"
)

type AMLRule struct {
	ID                 string          `json:"id" gorm:"primaryKey;type:varchar(100)"`
	Name               string          `json:"name" gorm:"type:varchar(255);not null"`
	Description        string          `json:"description" gorm:"type:text"`
	Category           AMLRuleCategory `json:"category" gorm:"type:varchar(50);not null"`
	Enabled            bool            `json:"enabled" gorm:"default:true"`
	Severity           AMLRuleSeverity `json:"severity" gorm:"type:varchar(20);not null"`
	Conditions         json.RawMessage `json:"conditions" gorm:"type:jsonb;not null"`
	Actions            json.RawMessage `json:"actions" gorm:"type:jsonb;not null"`
	Metadata           json.RawMessage `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	EffectivenessStats json.RawMessage `json:"effectiveness_stats" gorm:"type:jsonb;default:'{}'"`
	CreatedAt          time.Time       `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt          time.Time       `json:"updated_at" gorm:"not null;default:now()"`
	CreatedBy          *string         `json:"created_by" gorm:"type:uuid"`
	UpdatedBy          *string         `json:"updated_by" gorm:"type:uuid"`
}

func (AMLRule) TableName() string {
	return "aml_rules"
}

type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type RuleAction struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params"`
}

type VelocityRuleConditions struct {
	MaxTransactions int    `json:"max_transactions"`
	WindowHours     int    `json:"window_hours"`
	Field           string `json:"field"`
}

type EffectivenessStats struct {
	TotalTriggers  int `json:"total_triggers"`
	FalsePositives int `json:"false_positives"`
	SARConversions int `json:"sar_conversions"`
}
