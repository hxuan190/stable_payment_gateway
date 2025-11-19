-- Migration: Create payout_schedules table (PRD v2.2 - Advanced Off-ramp Strategies)
-- Purpose: Configure automatic payout scheduling (weekly/monthly) and threshold-based withdrawals

CREATE TABLE IF NOT EXISTS payout_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID UNIQUE NOT NULL, -- One schedule per merchant

    -- Scheduled Withdrawals Configuration
    scheduled_enabled BOOLEAN DEFAULT FALSE,
    scheduled_frequency VARCHAR(20), -- 'weekly', 'monthly', NULL if disabled
    scheduled_day_of_week INTEGER CHECK (scheduled_day_of_week BETWEEN 0 AND 6), -- 0=Sunday, 6=Saturday (for weekly)
    scheduled_day_of_month INTEGER CHECK (scheduled_day_of_month BETWEEN 1 AND 31), -- 1-31 (for monthly)
    scheduled_time TIME DEFAULT '10:00:00', -- Time of day to trigger (merchant's timezone)
    scheduled_timezone VARCHAR(50) DEFAULT 'Asia/Ho_Chi_Minh', -- IANA timezone identifier
    scheduled_withdraw_percentage INTEGER DEFAULT 80 CHECK (scheduled_withdraw_percentage BETWEEN 10 AND 100), -- % of available balance to withdraw

    -- Threshold-based Withdrawals Configuration
    threshold_enabled BOOLEAN DEFAULT FALSE,
    threshold_usdt DECIMAL(20, 8), -- Trigger when available balance >= this USDT amount
    threshold_withdraw_percentage INTEGER DEFAULT 90 CHECK (threshold_withdraw_percentage BETWEEN 10 AND 100), -- % to withdraw when threshold hit

    -- Minimum Withdrawal Amounts (safety limits)
    min_withdrawal_vnd DECIMAL(20, 2) DEFAULT 1000000, -- Min 1M VND per payout
    max_withdrawal_vnd DECIMAL(20, 2), -- Max per payout (NULL = no limit)

    -- Activity Tracking
    last_triggered_at TIMESTAMP, -- Last time a scheduled payout was triggered
    last_threshold_triggered_at TIMESTAMP, -- Last time threshold payout was triggered
    total_scheduled_payouts INTEGER DEFAULT 0,
    total_threshold_payouts INTEGER DEFAULT 0,

    -- Suspension (pause scheduled payouts temporarily)
    is_suspended BOOLEAN DEFAULT FALSE,
    suspended_at TIMESTAMP,
    suspension_reason TEXT,

    -- Metadata
    metadata JSONB DEFAULT '{}', -- Additional flexible configuration

    -- Audit fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(100), -- Admin/merchant who created the schedule
    updated_by VARCHAR(100)  -- Last updater
);

-- Indexes for payout_schedules
CREATE UNIQUE INDEX idx_payout_schedules_merchant ON payout_schedules(merchant_id);
CREATE INDEX idx_payout_schedules_scheduled_enabled ON payout_schedules(scheduled_enabled) WHERE scheduled_enabled = TRUE;
CREATE INDEX idx_payout_schedules_threshold_enabled ON payout_schedules(threshold_enabled) WHERE threshold_enabled = TRUE;
CREATE INDEX idx_payout_schedules_is_suspended ON payout_schedules(is_suspended) WHERE is_suspended = FALSE;
CREATE INDEX idx_payout_schedules_frequency ON payout_schedules(scheduled_frequency) WHERE scheduled_frequency IS NOT NULL;

-- Composite index for scheduler to find due payouts
CREATE INDEX idx_payout_schedules_due_check ON payout_schedules(scheduled_enabled, scheduled_frequency, scheduled_day_of_week, scheduled_day_of_month, is_suspended);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_payout_schedules_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_payout_schedules_updated_at
    BEFORE UPDATE ON payout_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_payout_schedules_updated_at();

-- Check constraints
ALTER TABLE payout_schedules ADD CONSTRAINT check_scheduled_frequency
    CHECK (scheduled_frequency IN ('weekly', 'monthly') OR scheduled_frequency IS NULL);

ALTER TABLE payout_schedules ADD CONSTRAINT check_scheduled_enabled_requires_frequency
    CHECK (NOT scheduled_enabled OR (scheduled_enabled AND scheduled_frequency IS NOT NULL));

ALTER TABLE payout_schedules ADD CONSTRAINT check_weekly_requires_day_of_week
    CHECK (scheduled_frequency != 'weekly' OR scheduled_day_of_week IS NOT NULL);

ALTER TABLE payout_schedules ADD CONSTRAINT check_monthly_requires_day_of_month
    CHECK (scheduled_frequency != 'monthly' OR scheduled_day_of_month IS NOT NULL);

ALTER TABLE payout_schedules ADD CONSTRAINT check_threshold_enabled_requires_amount
    CHECK (NOT threshold_enabled OR (threshold_enabled AND threshold_usdt IS NOT NULL));

ALTER TABLE payout_schedules ADD CONSTRAINT check_min_max_withdrawal
    CHECK (max_withdrawal_vnd IS NULL OR max_withdrawal_vnd >= min_withdrawal_vnd);

-- Function: Get next scheduled payout time for a merchant
CREATE OR REPLACE FUNCTION get_next_scheduled_payout(merchant_uuid UUID)
RETURNS TIMESTAMP AS $$
DECLARE
    schedule RECORD;
    next_payout TIMESTAMP;
BEGIN
    SELECT * INTO schedule
    FROM payout_schedules
    WHERE merchant_id = merchant_uuid
        AND scheduled_enabled = TRUE
        AND is_suspended = FALSE;

    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    -- Calculate next payout time based on frequency
    IF schedule.scheduled_frequency = 'weekly' THEN
        -- Find next occurrence of scheduled_day_of_week
        next_payout := date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE schedule.scheduled_timezone)
            + INTERVAL '1 day' * schedule.scheduled_day_of_week
            + schedule.scheduled_time;

        IF next_payout <= CURRENT_TIMESTAMP THEN
            next_payout := next_payout + INTERVAL '7 days';
        END IF;

    ELSIF schedule.scheduled_frequency = 'monthly' THEN
        -- Find next occurrence of scheduled_day_of_month
        next_payout := date_trunc('month', CURRENT_TIMESTAMP AT TIME ZONE schedule.scheduled_timezone)
            + INTERVAL '1 day' * (schedule.scheduled_day_of_month - 1)
            + schedule.scheduled_time;

        IF next_payout <= CURRENT_TIMESTAMP THEN
            next_payout := next_payout + INTERVAL '1 month';
        END IF;
    END IF;

    RETURN next_payout AT TIME ZONE schedule.scheduled_timezone AT TIME ZONE 'UTC';
END;
$$ LANGUAGE plpgsql;

-- View: Merchants with scheduled payouts due soon (next 24 hours)
CREATE OR REPLACE VIEW upcoming_scheduled_payouts AS
SELECT
    ps.merchant_id,
    ps.scheduled_frequency,
    ps.scheduled_day_of_week,
    ps.scheduled_day_of_month,
    ps.scheduled_time,
    ps.scheduled_withdraw_percentage,
    ps.last_triggered_at,
    get_next_scheduled_payout(ps.merchant_id) AS next_payout_time
FROM payout_schedules ps
WHERE ps.scheduled_enabled = TRUE
    AND ps.is_suspended = FALSE
    AND get_next_scheduled_payout(ps.merchant_id) <= CURRENT_TIMESTAMP + INTERVAL '24 hours';

-- View: Merchants with threshold-based payouts enabled
CREATE OR REPLACE VIEW threshold_payout_configs AS
SELECT
    ps.merchant_id,
    ps.threshold_usdt,
    ps.threshold_withdraw_percentage,
    ps.last_threshold_triggered_at,
    ps.min_withdrawal_vnd,
    ps.max_withdrawal_vnd
FROM payout_schedules ps
WHERE ps.threshold_enabled = TRUE
    AND ps.is_suspended = FALSE;

-- View: Payout schedule summary
CREATE OR REPLACE VIEW payout_schedule_summary AS
SELECT
    COUNT(*) AS total_merchants,
    COUNT(*) FILTER (WHERE scheduled_enabled = TRUE) AS scheduled_enabled_count,
    COUNT(*) FILTER (WHERE threshold_enabled = TRUE) AS threshold_enabled_count,
    COUNT(*) FILTER (WHERE is_suspended = TRUE) AS suspended_count,
    COUNT(*) FILTER (WHERE scheduled_frequency = 'weekly') AS weekly_count,
    COUNT(*) FILTER (WHERE scheduled_frequency = 'monthly') AS monthly_count,
    SUM(total_scheduled_payouts) AS total_scheduled_payouts_all_time,
    SUM(total_threshold_payouts) AS total_threshold_payouts_all_time
FROM payout_schedules;

-- Comments for documentation
COMMENT ON TABLE payout_schedules IS 'Merchant payout automation configuration: scheduled (weekly/monthly) and threshold-based withdrawals (PRD v2.2 Advanced Off-ramp)';
COMMENT ON COLUMN payout_schedules.scheduled_frequency IS 'Frequency of automatic payouts: weekly (every Monday, Tuesday, etc.) or monthly (1st, 15th, etc.)';
COMMENT ON COLUMN payout_schedules.scheduled_day_of_week IS 'For weekly schedules: 0=Sunday, 1=Monday, ..., 6=Saturday';
COMMENT ON COLUMN payout_schedules.scheduled_day_of_month IS 'For monthly schedules: day of month (1-31). If day > days in month, uses last day.';
COMMENT ON COLUMN payout_schedules.scheduled_withdraw_percentage IS 'Percentage of available balance to withdraw (10-100%)';
COMMENT ON COLUMN payout_schedules.threshold_usdt IS 'When available balance (in USDT equivalent) >= this, trigger automatic payout';
COMMENT ON COLUMN payout_schedules.threshold_withdraw_percentage IS 'Percentage to withdraw when threshold is hit';
COMMENT ON COLUMN payout_schedules.is_suspended IS 'Temporarily pause scheduled payouts (e.g., merchant on vacation)';
COMMENT ON FUNCTION get_next_scheduled_payout(UUID) IS 'Calculate next scheduled payout time for a merchant considering timezone';
COMMENT ON VIEW upcoming_scheduled_payouts IS 'Merchants with scheduled payouts due in next 24 hours';
COMMENT ON VIEW threshold_payout_configs IS 'Merchants with threshold-based auto-withdrawal enabled';
