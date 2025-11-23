-- Seed velocity check rules for AML compliance
-- This migration populates the aml_rules table with velocity-based transaction monitoring rules

-- Basic Velocity Rule: 10 transactions per 24 hours
INSERT INTO aml_rules (
    id,
    name,
    description,
    category,
    enabled,
    severity,
    conditions,
    actions,
    metadata,
    effectiveness_stats
) VALUES (
    'VEL_BASIC_001',
    'Basic Velocity Limit (10 tx/24h)',
    'Prevents spam and high-frequency trading by limiting wallet addresses to 10 transactions per 24 hours',
    'velocity',
    true,
    'MEDIUM',
    '{"max_transactions": 10, "window_hours": 24, "field": "tx_count_24h"}',
    '[{"type": "block_transaction", "params": {"reason": "velocity_limit_exceeded"}}, {"type": "create_alert", "params": {"alert_type": "VELOCITY_EXCEEDED", "notify_admin": false}}]',
    '{"evaluation_frequency": "per_transaction", "applies_to": "all_wallets", "version": "1.0"}',
    '{"total_triggers": 0, "false_positives": 0, "sar_conversions": 0}'
);

-- Velocity Spike Detection: 5x daily average
INSERT INTO aml_rules (
    id,
    name,
    description,
    category,
    enabled,
    severity,
    conditions,
    actions,
    metadata,
    effectiveness_stats
) VALUES (
    'VEL_SPIKE_001',
    'Transaction Count Spike (5x Average)',
    'Detects sudden spikes in transaction frequency (5x daily average) indicating potential laundering',
    'velocity',
    false,
    'HIGH',
    '{"spike_multiplier": 5, "baseline_window_days": 30, "detection_window_hours": 24, "field": "tx_count_spike"}',
    '[{"type": "require_review", "params": {"review_type": "manual_approval"}}, {"type": "create_alert", "params": {"alert_type": "VELOCITY_SPIKE", "notify_admin": true}}]',
    '{"evaluation_frequency": "per_transaction", "requires_baseline": true, "version": "1.0"}',
    '{"total_triggers": 0, "false_positives": 0, "sar_conversions": 0}'
);

-- Volume Spike Detection: 3x daily average
INSERT INTO aml_rules (
    id,
    name,
    description,
    category,
    enabled,
    severity,
    conditions,
    actions,
    metadata,
    effectiveness_stats
) VALUES (
    'VEL_VOLUME_001',
    'Transaction Volume Spike (3x Average)',
    'Detects sudden spikes in transaction volume (3x daily average USD amount)',
    'velocity',
    false,
    'HIGH',
    '{"spike_multiplier": 3, "baseline_window_days": 30, "detection_window_hours": 24, "field": "tx_volume_usd"}',
    '[{"type": "require_review", "params": {"review_type": "manual_approval"}}, {"type": "create_alert", "params": {"alert_type": "VOLUME_SPIKE", "notify_admin": true}}]',
    '{"evaluation_frequency": "per_transaction", "requires_baseline": true, "version": "1.0"}',
    '{"total_triggers": 0, "false_positives": 0, "sar_conversions": 0}'
);

-- High Frequency Trading Pattern
INSERT INTO aml_rules (
    id,
    name,
    description,
    category,
    enabled,
    severity,
    conditions,
    actions,
    metadata,
    effectiveness_stats
) VALUES (
    'VEL_HFT_001',
    'High Frequency Trading Pattern (5 tx/hour)',
    'Detects high-frequency trading patterns (5+ transactions within 1 hour)',
    'velocity',
    false,
    'MEDIUM',
    '{"max_transactions": 5, "window_hours": 1, "field": "tx_count_1h"}',
    '[{"type": "create_alert", "params": {"alert_type": "HFT_PATTERN", "notify_admin": false}}, {"type": "require_review", "params": {"review_type": "automated_check"}}]',
    '{"evaluation_frequency": "per_transaction", "applies_to": "all_wallets", "version": "1.0"}',
    '{"total_triggers": 0, "false_positives": 0, "sar_conversions": 0}'
);

-- Rapid Cash-Out Pattern
INSERT INTO aml_rules (
    id,
    name,
    description,
    category,
    enabled,
    severity,
    conditions,
    actions,
    metadata,
    effectiveness_stats
) VALUES (
    'VEL_CASHOUT_001',
    'Rapid Cash-Out Detection',
    'Detects rapid cash-out patterns (payout within 1 hour of payment)',
    'pattern',
    false,
    'HIGH',
    '{"max_time_to_payout_minutes": 60, "field": "time_to_payout"}',
    '[{"type": "require_review", "params": {"review_type": "manual_approval"}}, {"type": "create_alert", "params": {"alert_type": "RAPID_CASHOUT", "notify_admin": true}}]',
    '{"evaluation_frequency": "on_payout_request", "version": "1.0"}',
    '{"total_triggers": 0, "false_positives": 0, "sar_conversions": 0}'
);

COMMENT ON TABLE aml_rules IS 'Configurable AML monitoring rules - velocity checks now database-backed instead of hardcoded';

