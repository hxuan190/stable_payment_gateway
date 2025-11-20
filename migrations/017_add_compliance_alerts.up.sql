-- Add compliance_alerts table for tracking compliance issues
-- This table stores all compliance alerts that require manual review

CREATE TABLE IF NOT EXISTS compliance_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Alert classification
    alert_type VARCHAR(50) NOT NULL, -- 'sanctioned_address', 'high_risk_address', 'aml_failure', 'travel_rule_missing', etc.
    severity VARCHAR(20) NOT NULL DEFAULT 'medium', -- 'critical', 'high', 'medium', 'low'
    status VARCHAR(20) NOT NULL DEFAULT 'open', -- 'open', 'investigating', 'resolved', 'false_positive'

    -- Related entities
    payment_id UUID REFERENCES payments(id) ON DELETE SET NULL,
    merchant_id UUID REFERENCES merchants(id) ON DELETE SET NULL,
    payout_id UUID REFERENCES payouts(id) ON DELETE SET NULL,

    -- Alert details
    from_address TEXT, -- Wallet address that triggered the alert
    to_address TEXT,
    transaction_hash TEXT,
    blockchain VARCHAR(20), -- 'solana', 'bsc', 'ethereum', etc.

    -- Risk assessment
    risk_score INT CHECK (risk_score >= 0 AND risk_score <= 100),
    risk_flags TEXT[], -- Array of risk indicators

    -- Details and evidence
    details TEXT NOT NULL, -- Description of the compliance issue
    evidence JSONB, -- Additional evidence (screening results, TRM Labs data, etc.)

    -- Actions taken
    actions_taken TEXT[], -- Array of actions: 'payment_flagged', 'email_sent', 'funds_frozen', etc.
    recommended_action TEXT, -- What should be done

    -- Review workflow
    assigned_to VARCHAR(100), -- Email of ops team member
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMPTZ,
    resolution_notes TEXT,

    -- Notification tracking
    email_sent BOOLEAN DEFAULT false,
    email_sent_at TIMESTAMPTZ,
    email_recipients TEXT[], -- Array of email addresses notified

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,

    -- Indexes for common queries
    INDEX idx_compliance_alerts_status (status),
    INDEX idx_compliance_alerts_severity (severity),
    INDEX idx_compliance_alerts_payment_id (payment_id),
    INDEX idx_compliance_alerts_merchant_id (merchant_id),
    INDEX idx_compliance_alerts_type (alert_type),
    INDEX idx_compliance_alerts_created_at (created_at DESC),
    INDEX idx_compliance_alerts_from_address (from_address)
);

-- Add comments for documentation
COMMENT ON TABLE compliance_alerts IS 'Stores compliance alerts that require manual review by ops team';
COMMENT ON COLUMN compliance_alerts.alert_type IS 'Type of compliance issue: sanctioned_address, high_risk_address, aml_failure, etc.';
COMMENT ON COLUMN compliance_alerts.severity IS 'Severity level: critical (immediate action), high (same day), medium (24h), low (review when possible)';
COMMENT ON COLUMN compliance_alerts.status IS 'Current status: open (needs review), investigating (in progress), resolved (handled), false_positive (no action needed)';
COMMENT ON COLUMN compliance_alerts.actions_taken IS 'Array of actions taken: payment_flagged, email_sent, funds_frozen, merchant_notified, law_enforcement_contacted';

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_compliance_alerts_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER compliance_alerts_updated_at
    BEFORE UPDATE ON compliance_alerts
    FOR EACH ROW
    EXECUTE FUNCTION update_compliance_alerts_updated_at();
