-- Migration 018: Create OTC Partner Liquidity Monitoring Table
-- Purpose: Track OTC partner VND pool levels to prevent payment failures
-- Alert when balance falls below 50M VND

CREATE TABLE IF NOT EXISTS otc_partner_liquidity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_name VARCHAR(100) NOT NULL,
    partner_code VARCHAR(50) NOT NULL UNIQUE,

    -- Balance tracking
    current_balance_vnd DECIMAL(20, 2) NOT NULL DEFAULT 0,
    available_balance_vnd DECIMAL(20, 2) NOT NULL DEFAULT 0, -- Balance minus pending payouts
    reserved_balance_vnd DECIMAL(20, 2) NOT NULL DEFAULT 0,  -- Amount reserved for pending payouts

    -- Balance thresholds
    low_balance_threshold_vnd DECIMAL(20, 2) NOT NULL DEFAULT 50000000, -- 50M VND
    critical_threshold_vnd DECIMAL(20, 2) NOT NULL DEFAULT 10000000,     -- 10M VND

    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    is_balance_low BOOLEAN NOT NULL DEFAULT FALSE,
    is_balance_critical BOOLEAN NOT NULL DEFAULT FALSE,
    last_alert_sent_at TIMESTAMP,
    alert_count_today INTEGER NOT NULL DEFAULT 0,

    -- Partner details
    api_endpoint VARCHAR(500),
    api_key_encrypted TEXT, -- Store encrypted API key for balance checks
    contact_person VARCHAR(255),
    contact_email VARCHAR(255),
    contact_phone VARCHAR(50),

    -- Operational hours
    operating_hours_start TIME DEFAULT '08:00:00',
    operating_hours_end TIME DEFAULT '18:00:00',
    timezone VARCHAR(50) DEFAULT 'Asia/Ho_Chi_Minh',
    is_24_7 BOOLEAN DEFAULT FALSE,

    -- Audit fields
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_balance_check_at TIMESTAMP,
    last_balance_update_at TIMESTAMP,

    -- Metadata
    metadata JSONB,
    notes TEXT,

    CONSTRAINT check_status CHECK (status IN ('active', 'inactive', 'suspended', 'maintenance')),
    CONSTRAINT check_balances CHECK (
        current_balance_vnd >= 0 AND
        available_balance_vnd >= 0 AND
        reserved_balance_vnd >= 0 AND
        available_balance_vnd = current_balance_vnd - reserved_balance_vnd
    )
);

-- Create balance history table for auditing
CREATE TABLE IF NOT EXISTS otc_partner_liquidity_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_id UUID NOT NULL REFERENCES otc_partner_liquidity(id) ON DELETE CASCADE,

    -- Balance snapshot
    previous_balance_vnd DECIMAL(20, 2) NOT NULL,
    new_balance_vnd DECIMAL(20, 2) NOT NULL,
    balance_change_vnd DECIMAL(20, 2) NOT NULL,

    -- Change reason
    change_type VARCHAR(50) NOT NULL,
    change_reason TEXT,
    reference_id UUID, -- Payment ID, Payout ID, etc.

    -- Timestamp
    recorded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_change_type CHECK (change_type IN (
        'manual_update',
        'api_sync',
        'payout_reserved',
        'payout_completed',
        'payout_cancelled',
        'top_up',
        'adjustment'
    ))
);

-- Create alert log table
CREATE TABLE IF NOT EXISTS otc_liquidity_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_id UUID NOT NULL REFERENCES otc_partner_liquidity(id) ON DELETE CASCADE,

    -- Alert details
    alert_type VARCHAR(50) NOT NULL,
    alert_level VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    current_balance_vnd DECIMAL(20, 2) NOT NULL,
    threshold_vnd DECIMAL(20, 2) NOT NULL,

    -- Notification tracking
    sent_to_telegram BOOLEAN DEFAULT FALSE,
    sent_to_slack BOOLEAN DEFAULT FALSE,
    sent_to_email BOOLEAN DEFAULT FALSE,

    -- Resolution tracking
    is_resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP,
    resolved_by VARCHAR(100),
    resolution_notes TEXT,

    -- Timestamp
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_alert_type CHECK (alert_type IN ('low_balance', 'critical_balance', 'offline', 'api_error')),
    CONSTRAINT check_alert_level CHECK (alert_level IN ('warning', 'critical', 'error'))
);

-- Create indexes
CREATE INDEX idx_otc_partner_liquidity_status ON otc_partner_liquidity(status);
CREATE INDEX idx_otc_partner_liquidity_balance ON otc_partner_liquidity(current_balance_vnd);
CREATE INDEX idx_otc_partner_liquidity_alerts ON otc_partner_liquidity(is_balance_low, is_balance_critical);
CREATE INDEX idx_otc_partner_liquidity_updated ON otc_partner_liquidity(updated_at);

CREATE INDEX idx_otc_liquidity_history_partner ON otc_partner_liquidity_history(partner_id, recorded_at DESC);
CREATE INDEX idx_otc_liquidity_history_recorded ON otc_partner_liquidity_history(recorded_at DESC);

CREATE INDEX idx_otc_alerts_partner ON otc_liquidity_alerts(partner_id, created_at DESC);
CREATE INDEX idx_otc_alerts_unresolved ON otc_liquidity_alerts(is_resolved, created_at DESC) WHERE is_resolved = FALSE;
CREATE INDEX idx_otc_alerts_created ON otc_liquidity_alerts(created_at DESC);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_otc_partner_liquidity_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_otc_partner_liquidity_updated_at
    BEFORE UPDATE ON otc_partner_liquidity
    FOR EACH ROW
    EXECUTE FUNCTION update_otc_partner_liquidity_updated_at();

-- Create trigger to auto-update balance flags
CREATE OR REPLACE FUNCTION update_otc_balance_flags()
RETURNS TRIGGER AS $$
BEGIN
    -- Update low balance flag
    NEW.is_balance_low := NEW.current_balance_vnd < NEW.low_balance_threshold_vnd;

    -- Update critical balance flag
    NEW.is_balance_critical := NEW.current_balance_vnd < NEW.critical_threshold_vnd;

    -- Update available balance
    NEW.available_balance_vnd := NEW.current_balance_vnd - NEW.reserved_balance_vnd;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_otc_balance_flags
    BEFORE INSERT OR UPDATE OF current_balance_vnd, reserved_balance_vnd, low_balance_threshold_vnd, critical_threshold_vnd
    ON otc_partner_liquidity
    FOR EACH ROW
    EXECUTE FUNCTION update_otc_balance_flags();

-- Add comments
COMMENT ON TABLE otc_partner_liquidity IS 'OTC partner VND liquidity pool tracking and monitoring';
COMMENT ON COLUMN otc_partner_liquidity.current_balance_vnd IS 'Current total VND balance in partner pool';
COMMENT ON COLUMN otc_partner_liquidity.available_balance_vnd IS 'Available VND balance (current - reserved)';
COMMENT ON COLUMN otc_partner_liquidity.reserved_balance_vnd IS 'VND reserved for pending payouts';
COMMENT ON COLUMN otc_partner_liquidity.low_balance_threshold_vnd IS 'Alert threshold for low balance (default 50M VND)';
COMMENT ON COLUMN otc_partner_liquidity.critical_threshold_vnd IS 'Alert threshold for critical balance (default 10M VND)';

COMMENT ON TABLE otc_partner_liquidity_history IS 'Historical record of all OTC partner balance changes';
COMMENT ON TABLE otc_liquidity_alerts IS 'Log of all liquidity alerts sent to ops team';

-- Insert default OTC partners (example data)
INSERT INTO otc_partner_liquidity (partner_name, partner_code, current_balance_vnd, low_balance_threshold_vnd, critical_threshold_vnd, contact_email)
VALUES
    ('Primary OTC Partner', 'PRIMARY_OTC', 500000000, 50000000, 10000000, 'otc@partner.vn'),
    ('Backup OTC Partner', 'BACKUP_OTC', 200000000, 30000000, 5000000, 'backup@partner.vn')
ON CONFLICT (partner_code) DO NOTHING;
