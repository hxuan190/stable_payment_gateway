-- Migration: Create AML (Anti-Money Laundering) Engine Tables
-- Version: 005
-- Description: Comprehensive AML compliance system including risk scoring,
--              transaction monitoring, alerts, cases, sanctions screening, and reporting
-- Created: 2025-11-19

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- AML Customer Risk Scores
-- ============================================================================
-- Stores risk assessment for each merchant/customer
-- Risk levels: low (0-30), medium (31-60), high (61-85), prohibited (86-100)
-- ============================================================================

CREATE TABLE aml_customer_risk_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    risk_level VARCHAR(20) NOT NULL CHECK (risk_level IN ('low', 'medium', 'high', 'prohibited')),
    risk_score INTEGER NOT NULL CHECK (risk_score >= 0 AND risk_score <= 100),
    risk_factors JSONB NOT NULL DEFAULT '{}', -- Detailed breakdown of risk factors
    last_assessed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    next_review_date DATE NOT NULL, -- Periodic review based on risk level
    assessed_by VARCHAR(50) NOT NULL DEFAULT 'system', -- 'system' or admin user ID
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_aml_risk_merchant ON aml_customer_risk_scores(merchant_id);
CREATE INDEX idx_aml_risk_level ON aml_customer_risk_scores(risk_level);
CREATE INDEX idx_aml_risk_review ON aml_customer_risk_scores(next_review_date);
CREATE INDEX idx_aml_risk_score ON aml_customer_risk_scores(risk_score DESC);

-- Auto-update updated_at timestamp
CREATE TRIGGER update_aml_risk_updated_at
    BEFORE UPDATE ON aml_customer_risk_scores
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE aml_customer_risk_scores IS 'Customer risk assessment scores and factors for AML compliance';
COMMENT ON COLUMN aml_customer_risk_scores.risk_factors IS 'JSON object containing detailed risk factor breakdown (e.g., {"business_type": 10, "kyc_complete": -5, "transaction_volume": 25})';
COMMENT ON COLUMN aml_customer_risk_scores.next_review_date IS 'Next scheduled review date (low risk: annual, medium: semi-annual, high: quarterly)';

-- ============================================================================
-- AML Transaction Monitoring
-- ============================================================================
-- Real-time and batch monitoring of all payment transactions
-- Stores monitoring results, risk scores, and triggered rules
-- ============================================================================

CREATE TABLE aml_transaction_monitoring (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID REFERENCES payments(id) ON DELETE CASCADE,
    merchant_id UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    amount_vnd DECIMAL(20, 2) NOT NULL,
    amount_crypto DECIMAL(20, 8),
    crypto_currency VARCHAR(10),
    monitoring_status VARCHAR(20) NOT NULL DEFAULT 'clean' CHECK (monitoring_status IN ('clean', 'flagged', 'blocked', 'reviewing')),
    risk_score INTEGER CHECK (risk_score >= 0 AND risk_score <= 100), -- Transaction-specific risk score
    rules_triggered TEXT[] DEFAULT '{}', -- Array of rule IDs that fired
    wallet_address VARCHAR(255),
    wallet_risk_score INTEGER CHECK (wallet_risk_score >= 0 AND wallet_risk_score <= 100),
    metadata JSONB DEFAULT '{}', -- Additional context (e.g., velocity metrics, baseline comparison)
    reviewed BOOLEAN DEFAULT FALSE,
    reviewed_by UUID, -- REFERENCES admin_users(id) - will be added when admin_users table exists
    reviewed_at TIMESTAMP,
    review_notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for querying and reporting
CREATE INDEX idx_aml_txn_payment ON aml_transaction_monitoring(payment_id);
CREATE INDEX idx_aml_txn_merchant ON aml_transaction_monitoring(merchant_id);
CREATE INDEX idx_aml_txn_status ON aml_transaction_monitoring(monitoring_status);
CREATE INDEX idx_aml_txn_reviewed ON aml_transaction_monitoring(reviewed) WHERE reviewed = FALSE;
CREATE INDEX idx_aml_txn_created ON aml_transaction_monitoring(created_at DESC);
CREATE INDEX idx_aml_txn_wallet ON aml_transaction_monitoring(wallet_address) WHERE wallet_address IS NOT NULL;
CREATE INDEX idx_aml_txn_amount ON aml_transaction_monitoring(amount_vnd DESC);

COMMENT ON TABLE aml_transaction_monitoring IS 'Transaction-level AML monitoring and risk assessment';
COMMENT ON COLUMN aml_transaction_monitoring.rules_triggered IS 'Array of rule IDs that triggered for this transaction (e.g., ["THRESHOLD_VN_001", "VEL_003"])';
COMMENT ON COLUMN aml_transaction_monitoring.metadata IS 'Additional monitoring context: velocity metrics, baseline comparison, behavioral analysis';

-- ============================================================================
-- AML Alerts
-- ============================================================================
-- Alert management system for suspicious activities
-- Supports workflow: created → assigned → under_review → resolved
-- ============================================================================

CREATE TABLE aml_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_type VARCHAR(50) NOT NULL, -- 'THRESHOLD', 'PATTERN', 'SANCTIONS', 'WALLET', 'VELOCITY', etc.
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    merchant_id UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    payment_id UUID REFERENCES payments(id) ON DELETE SET NULL,
    rule_id VARCHAR(100), -- Reference to aml_rules.id
    rule_name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    metadata JSONB DEFAULT '{}', -- Rule-specific data (e.g., threshold exceeded, pattern details)
    status VARCHAR(30) NOT NULL DEFAULT 'created' CHECK (status IN ('created', 'assigned', 'under_review', 'resolved')),
    assigned_to UUID, -- REFERENCES admin_users(id)
    resolution VARCHAR(30) CHECK (resolution IN ('cleared', 'false_positive', 'escalated', 'sar_filed', 'account_suspended')),
    resolution_notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    assigned_at TIMESTAMP,
    reviewed_at TIMESTAMP,
    resolved_at TIMESTAMP
);

-- Indexes for alert management dashboard
CREATE INDEX idx_aml_alert_merchant ON aml_alerts(merchant_id);
CREATE INDEX idx_aml_alert_payment ON aml_alerts(payment_id) WHERE payment_id IS NOT NULL;
CREATE INDEX idx_aml_alert_status ON aml_alerts(status);
CREATE INDEX idx_aml_alert_severity ON aml_alerts(severity);
CREATE INDEX idx_aml_alert_assigned ON aml_alerts(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_aml_alert_created ON aml_alerts(created_at DESC);
CREATE INDEX idx_aml_alert_type ON aml_alerts(alert_type);
CREATE INDEX idx_aml_alert_unresolved ON aml_alerts(severity, created_at) WHERE status != 'resolved';

COMMENT ON TABLE aml_alerts IS 'AML alert management system for suspicious activity detection and review';
COMMENT ON COLUMN aml_alerts.metadata IS 'Alert-specific details: amounts, counts, patterns, comparisons to baseline';
COMMENT ON COLUMN aml_alerts.resolution IS 'Final outcome after review: cleared (no issue), false_positive, escalated to case, SAR filed, or account action taken';

-- ============================================================================
-- AML Cases
-- ============================================================================
-- Case management for complex investigations
-- Aggregates multiple alerts and provides investigation workflow
-- ============================================================================

-- Create sequence for human-readable case numbers
CREATE SEQUENCE aml_case_number_seq START 1;

CREATE TABLE aml_cases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_number VARCHAR(50) UNIQUE NOT NULL DEFAULT 'CASE-' || TO_CHAR(NOW(), 'YYYY') || '-' || LPAD(nextval('aml_case_number_seq')::TEXT, 6, '0'),
    merchant_id UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    case_type VARCHAR(50) NOT NULL CHECK (case_type IN ('investigation', 'sar_preparation', 'periodic_review', 'external_request')),
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    status VARCHAR(30) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'investigating', 'pending_decision', 'closed')),
    alert_ids UUID[] DEFAULT '{}', -- Related alerts
    assigned_to UUID, -- REFERENCES admin_users(id) - compliance officer
    opened_at TIMESTAMP NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMP,
    outcome VARCHAR(50) CHECK (outcome IN ('cleared', 'sar_filed', 'sar_declined', 'account_suspended', 'account_terminated', 'referred_to_authorities')),
    summary TEXT,
    investigation_notes TEXT,
    supporting_documents JSONB DEFAULT '{}', -- File paths/URLs to evidence: {"kyc_docs": ["path1"], "screenshots": ["path2"]}
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for case management
CREATE INDEX idx_aml_case_merchant ON aml_cases(merchant_id);
CREATE INDEX idx_aml_case_status ON aml_cases(status);
CREATE INDEX idx_aml_case_assigned ON aml_cases(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_aml_case_opened ON aml_cases(opened_at DESC);
CREATE INDEX idx_aml_case_number ON aml_cases(case_number);
CREATE INDEX idx_aml_case_type ON aml_cases(case_type);

-- Auto-update updated_at timestamp
CREATE TRIGGER update_aml_case_updated_at
    BEFORE UPDATE ON aml_cases
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE aml_cases IS 'Case management for complex AML investigations';
COMMENT ON COLUMN aml_cases.case_number IS 'Human-readable case identifier (e.g., CASE-2025-000001)';
COMMENT ON COLUMN aml_cases.alert_ids IS 'Array of alert UUIDs related to this case';
COMMENT ON COLUMN aml_cases.supporting_documents IS 'JSON object mapping document types to file paths';

-- ============================================================================
-- AML Sanctions List
-- ============================================================================
-- Stores global sanctions list entries for screening
-- Sources: OFAC (US), UN, EU, Interpol, Vietnam government
-- ============================================================================

CREATE TABLE aml_sanctions_list (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    list_source VARCHAR(50) NOT NULL CHECK (list_source IN ('OFAC', 'UN', 'EU', 'INTERPOL', 'VIETNAM', 'OTHER')),
    entry_type VARCHAR(30) NOT NULL CHECK (entry_type IN ('individual', 'entity', 'vessel', 'aircraft')),
    name_primary VARCHAR(500) NOT NULL,
    name_aliases TEXT[] DEFAULT '{}', -- Alternative names, spellings
    id_numbers TEXT[] DEFAULT '{}', -- Passport, national ID, tax ID, etc.
    addresses TEXT[] DEFAULT '{}',
    date_of_birth DATE,
    place_of_birth VARCHAR(255),
    nationalities VARCHAR(10)[] DEFAULT '{}', -- ISO country codes
    programs TEXT[] DEFAULT '{}', -- Sanction programs (e.g., ['IRGQ', 'SDGT'])
    remarks TEXT,
    list_updated_at DATE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for fast sanctions screening (fuzzy name matching)
CREATE INDEX idx_sanctions_name ON aml_sanctions_list USING gin(to_tsvector('simple', name_primary));
CREATE INDEX idx_sanctions_name_lower ON aml_sanctions_list(LOWER(name_primary));
CREATE INDEX idx_sanctions_source ON aml_sanctions_list(list_source);
CREATE INDEX idx_sanctions_type ON aml_sanctions_list(entry_type);
CREATE INDEX idx_sanctions_updated ON aml_sanctions_list(list_updated_at DESC);

-- Full-text search on aliases
CREATE INDEX idx_sanctions_aliases ON aml_sanctions_list USING gin(name_aliases);

-- Auto-update updated_at timestamp
CREATE TRIGGER update_aml_sanctions_updated_at
    BEFORE UPDATE ON aml_sanctions_list
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE aml_sanctions_list IS 'Global sanctions list aggregation for customer and wallet screening';
COMMENT ON COLUMN aml_sanctions_list.programs IS 'Specific sanction programs individual/entity is listed under';
COMMENT ON COLUMN aml_sanctions_list.list_updated_at IS 'Date when source list was last updated (for tracking data freshness)';

-- ============================================================================
-- AML Wallet Screening
-- ============================================================================
-- Blockchain wallet risk assessment and sanctions screening
-- Integrates with external APIs (Chainalysis, TRM Labs, Elliptic)
-- ============================================================================

CREATE TABLE aml_wallet_screening (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_address VARCHAR(255) NOT NULL,
    blockchain VARCHAR(20) NOT NULL CHECK (blockchain IN ('solana', 'bsc', 'ethereum', 'polygon', 'bitcoin')),
    risk_level VARCHAR(20) NOT NULL CHECK (risk_level IN ('low', 'medium', 'high', 'prohibited')),
    risk_score INTEGER NOT NULL CHECK (risk_score >= 0 AND risk_score <= 100),
    risk_factors JSONB NOT NULL DEFAULT '{}', -- Detailed risk breakdown: mixer usage, darknet links, etc.
    is_sanctioned BOOLEAN DEFAULT FALSE,
    sanction_details JSONB, -- Details if sanctioned: {"list": "OFAC", "program": "SDGT", "date_added": "2024-01-01"}
    screening_source VARCHAR(50), -- 'chainalysis', 'trm', 'elliptic', 'internal'
    last_screened_at TIMESTAMP NOT NULL DEFAULT NOW(),
    cache_expires_at TIMESTAMP NOT NULL DEFAULT NOW() + INTERVAL '7 days', -- Cache for 7 days
    metadata JSONB DEFAULT '{}', -- Additional API response data
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Unique index for wallet + chain (one screening per wallet per chain)
CREATE UNIQUE INDEX idx_wallet_screen_addr_chain ON aml_wallet_screening(wallet_address, blockchain);
CREATE INDEX idx_wallet_screen_risk ON aml_wallet_screening(risk_level);
CREATE INDEX idx_wallet_screen_sanctioned ON aml_wallet_screening(is_sanctioned) WHERE is_sanctioned = TRUE;
CREATE INDEX idx_wallet_screen_expires ON aml_wallet_screening(cache_expires_at);
CREATE INDEX idx_wallet_screen_updated ON aml_wallet_screening(last_screened_at DESC);

-- Auto-update updated_at timestamp
CREATE TRIGGER update_aml_wallet_updated_at
    BEFORE UPDATE ON aml_wallet_screening
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE aml_wallet_screening IS 'Blockchain wallet risk assessment and sanctions screening cache';
COMMENT ON COLUMN aml_wallet_screening.risk_factors IS 'Detailed risk breakdown: {"mixer": 40, "darknet": 0, "indirect_mixer": 20, "wallet_age_days": 10}';
COMMENT ON COLUMN aml_wallet_screening.cache_expires_at IS 'Cached results expire after 7 days, requiring re-screening';

-- ============================================================================
-- AML Rules
-- ============================================================================
-- Configurable rule engine for transaction monitoring
-- Allows dynamic rule management without code changes
-- ============================================================================

CREATE TABLE aml_rules (
    id VARCHAR(100) PRIMARY KEY, -- e.g., 'THRESHOLD_VN_001'
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL CHECK (category IN ('threshold', 'pattern', 'velocity', 'behavioral', 'sanctions', 'wallet')),
    enabled BOOLEAN DEFAULT TRUE,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    conditions JSONB NOT NULL, -- Rule conditions in structured format
    actions JSONB NOT NULL, -- Actions to take when rule triggers
    metadata JSONB DEFAULT '{}', -- Additional config (e.g., evaluation frequency)
    effectiveness_stats JSONB DEFAULT '{}', -- Track: triggers, false_positives, sar_conversions
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by UUID, -- REFERENCES admin_users(id)
    updated_by UUID -- REFERENCES admin_users(id)
);

-- Indexes for rule management
CREATE INDEX idx_aml_rule_category ON aml_rules(category);
CREATE INDEX idx_aml_rule_enabled ON aml_rules(enabled) WHERE enabled = TRUE;
CREATE INDEX idx_aml_rule_severity ON aml_rules(severity);

-- Auto-update updated_at timestamp
CREATE TRIGGER update_aml_rule_updated_at
    BEFORE UPDATE ON aml_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE aml_rules IS 'Configurable AML monitoring rules for transaction pattern detection';
COMMENT ON COLUMN aml_rules.conditions IS 'JSON array of conditions: [{"field": "amount_vnd", "operator": ">=", "value": 400000000}]';
COMMENT ON COLUMN aml_rules.actions IS 'JSON array of actions: [{"type": "create_alert", "params": {"alert_type": "THRESHOLD"}}]';
COMMENT ON COLUMN aml_rules.effectiveness_stats IS 'Track rule performance: {"total_triggers": 150, "false_positives": 20, "sar_filed": 2}';

-- ============================================================================
-- AML Reports
-- ============================================================================
-- Regulatory and internal AML reporting
-- Stores generated reports (SAR, threshold reports, monthly summaries)
-- ============================================================================

CREATE TABLE aml_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_type VARCHAR(50) NOT NULL CHECK (report_type IN ('SAR', 'THRESHOLD', 'DAILY_SUMMARY', 'MONTHLY', 'QUARTERLY', 'ANNUAL')),
    report_period_start DATE,
    report_period_end DATE,
    generated_by UUID, -- REFERENCES admin_users(id)
    generated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    file_path VARCHAR(500), -- Path to generated PDF/CSV/Excel file
    file_format VARCHAR(10) CHECK (file_format IN ('PDF', 'CSV', 'XLSX', 'JSON')),
    status VARCHAR(30) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'finalized', 'submitted', 'archived')),
    submitted_to VARCHAR(100), -- 'State Bank of Vietnam', 'Internal', 'Authorities'
    submitted_at TIMESTAMP,
    submission_reference VARCHAR(100), -- External reference number from regulatory authority
    metadata JSONB DEFAULT '{}', -- Report-specific data (e.g., total transactions, alert counts)
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for report management
CREATE INDEX idx_aml_report_type ON aml_reports(report_type);
CREATE INDEX idx_aml_report_status ON aml_reports(status);
CREATE INDEX idx_aml_report_period ON aml_reports(report_period_start, report_period_end);
CREATE INDEX idx_aml_report_generated ON aml_reports(generated_at DESC);
CREATE INDEX idx_aml_report_submitted ON aml_reports(submitted_at) WHERE submitted_at IS NOT NULL;

-- Auto-update updated_at timestamp
CREATE TRIGGER update_aml_report_updated_at
    BEFORE UPDATE ON aml_reports
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE aml_reports IS 'AML regulatory and internal reporting system';
COMMENT ON COLUMN aml_reports.metadata IS 'Report summary data: {"total_transactions": 1500, "alerts_generated": 45, "sar_filed": 2}';
COMMENT ON COLUMN aml_reports.submission_reference IS 'Reference number from regulatory authority after submission';

-- ============================================================================
-- AML Audit Log
-- ============================================================================
-- Comprehensive audit trail for all AML system activities
-- Immutable log for compliance and forensic analysis
-- ============================================================================

CREATE TABLE aml_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_type VARCHAR(30) NOT NULL CHECK (actor_type IN ('system', 'admin', 'compliance_officer', 'api')),
    actor_id UUID, -- Admin user ID if applicable, NULL for system
    action VARCHAR(100) NOT NULL, -- 'alert_created', 'case_opened', 'rule_updated', 'sanctions_screened', etc.
    resource_type VARCHAR(50) NOT NULL, -- 'alert', 'case', 'rule', 'merchant', 'payment', 'wallet'
    resource_id UUID,
    changes JSONB, -- Before/after state for updates: {"before": {...}, "after": {...}}
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for audit log queries
CREATE INDEX idx_aml_audit_actor ON aml_audit_log(actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_aml_audit_actor_type ON aml_audit_log(actor_type);
CREATE INDEX idx_aml_audit_resource ON aml_audit_log(resource_type, resource_id);
CREATE INDEX idx_aml_audit_created ON aml_audit_log(created_at DESC);
CREATE INDEX idx_aml_audit_action ON aml_audit_log(action);

-- Partition by month for performance (optional, implement when volume grows)
-- CREATE TABLE aml_audit_log_y2025m11 PARTITION OF aml_audit_log
--     FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');

COMMENT ON TABLE aml_audit_log IS 'Immutable audit trail for all AML system activities';
COMMENT ON COLUMN aml_audit_log.changes IS 'Change tracking for updates: {"field": {"old": "value1", "new": "value2"}}';

-- ============================================================================
-- Helper Functions
-- ============================================================================

-- Function to auto-update updated_at column (if not exists from previous migrations)
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Initial Data: Seed Default AML Rules
-- ============================================================================

INSERT INTO aml_rules (id, name, description, category, enabled, severity, conditions, actions, metadata) VALUES
-- Threshold Rules
('THRESHOLD_VN_001',
 'Vietnam Legal Threshold',
 'Vietnam requires reporting of transactions ≥ 400M VND',
 'threshold',
 TRUE,
 'MEDIUM',
 '[{"field": "amount_vnd", "operator": ">=", "value": 400000000}]',
 '[{"type": "create_alert", "params": {"alert_type": "THRESHOLD"}}, {"type": "flag_for_reporting", "params": {"report_type": "threshold_report"}}]',
 '{"legal_requirement": true, "jurisdiction": "VN", "source": "Decree 74/2023/ND-CP"}'
),

('THRESHOLD_CRYPTO_001',
 'High-Value Crypto Transaction',
 'Internal threshold for crypto payments (10M VND ≈ $400 USD)',
 'threshold',
 TRUE,
 'LOW',
 '[{"field": "amount_vnd", "operator": ">=", "value": 10000000}]',
 '[{"type": "create_alert", "params": {"alert_type": "THRESHOLD"}}]',
 '{"internal_threshold": true}'
),

('THRESHOLD_DAILY_001',
 'Daily Aggregate Threshold',
 'Flag merchants exceeding 1B VND in single day',
 'threshold',
 TRUE,
 'MEDIUM',
 '[{"field": "daily_total_vnd", "operator": ">=", "value": 1000000000}]',
 '[{"type": "create_alert", "params": {"alert_type": "THRESHOLD_AGGREGATE"}}]',
 '{"aggregate_type": "daily"}'
),

-- Structuring Rules
('STRUCT_001',
 'Structuring Detection - 24 Hour Window',
 'Multiple transactions just below threshold within 24 hours (potential smurfing)',
 'pattern',
 TRUE,
 'MEDIUM',
 '[{"field": "tx_count_24h", "operator": ">=", "value": 3}, {"field": "amount_vnd", "operator": "between", "value": [8000000, 9500000]}, {"field": "same_merchant", "operator": "==", "value": true}]',
 '[{"type": "create_alert", "params": {"alert_type": "STRUCTURING"}}]',
 '{"window_hours": 24, "threshold_range": "80-95%"}'
),

-- Velocity Rules
('VEL_001',
 'Transaction Velocity Spike',
 'Transaction count today > 5x daily average (for established merchants)',
 'velocity',
 TRUE,
 'MEDIUM',
 '[{"field": "merchant_age_days", "operator": ">", "value": 30}, {"field": "tx_count_today", "operator": ">", "value": "5x_avg"}]',
 '[{"type": "create_alert", "params": {"alert_type": "VELOCITY_SPIKE"}}]',
 '{"baseline_period": "30_days", "multiplier": 5}'
),

('VEL_002',
 'Volume Spike',
 'Daily volume > 3x average',
 'velocity',
 TRUE,
 'MEDIUM',
 '[{"field": "merchant_age_days", "operator": ">", "value": 30}, {"field": "daily_volume_vnd", "operator": ">", "value": "3x_avg"}]',
 '[{"type": "create_alert", "params": {"alert_type": "VOLUME_SPIKE"}}]',
 '{"baseline_period": "30_days", "multiplier": 3}'
),

-- Rapid Cash-Out
('RAPID_001',
 'Rapid Cash-Out',
 'Payout request within 1 hour of payment confirmation (>80% of balance)',
 'pattern',
 TRUE,
 'HIGH',
 '[{"field": "time_since_payment_minutes", "operator": "<=", "value": 60}, {"field": "payout_percentage", "operator": ">=", "value": 80}]',
 '[{"type": "create_alert", "params": {"alert_type": "RAPID_CASHOUT"}}, {"type": "require_review", "params": {"review_type": "enhanced"}}]',
 '{"time_window_minutes": 60, "payout_threshold_pct": 80}'
),

-- Round Amount Detection
('ROUND_001',
 'Suspicious Round Amounts',
 'Exact round numbers may indicate suspicious activity',
 'pattern',
 TRUE,
 'LOW',
 '[{"field": "amount_vnd", "operator": "in", "value": [1000000, 5000000, 10000000, 50000000, 100000000]}]',
 '[{"type": "create_alert", "params": {"alert_type": "ROUND_AMOUNT"}}]',
 '{"round_amounts": [1000000, 5000000, 10000000, 50000000, 100000000]}'
);

-- ============================================================================
-- Grants (adjust based on your user roles)
-- ============================================================================

-- Grant permissions to application user (adjust username as needed)
-- GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO payment_gateway_app;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO payment_gateway_app;

-- ============================================================================
-- Migration Complete
-- ============================================================================

COMMENT ON SCHEMA public IS 'AML Engine tables created successfully - Version 005';
