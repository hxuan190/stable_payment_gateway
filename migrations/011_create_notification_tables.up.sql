-- Migration: Create notification tables (PRD v2.2 - Omni-channel Notification Center)
-- Purpose: Track notification delivery across multiple channels

-- Table: notification_logs - Track all notification attempts
CREATE TABLE IF NOT EXISTS notification_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Target Information
    payment_id UUID, -- Reference to payment (nullable for system notifications)
    merchant_id UUID NOT NULL, -- Reference to merchant

    -- Notification Details
    channel VARCHAR(50) NOT NULL, -- 'email', 'telegram', 'zalo', 'webhook', 'speaker'
    event_type VARCHAR(50) NOT NULL, -- 'payment_confirmed', 'payment_expired', 'payout_approved', etc.
    recipient VARCHAR(255), -- Email address, Telegram chat ID, phone number, etc.

    -- Delivery Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'sent', 'delivered', 'failed', 'retrying'
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    failed_at TIMESTAMP,
    error_message TEXT,
    error_code VARCHAR(50),

    -- Retry Logic
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    next_retry_at TIMESTAMP,

    -- Message Content
    subject VARCHAR(500), -- For email
    message TEXT, -- Full message content
    template_id VARCHAR(100), -- Template identifier used

    -- Metadata
    provider_message_id VARCHAR(255), -- External provider's message ID
    provider_response JSONB, -- Full response from notification provider
    metadata JSONB DEFAULT '{}', -- Additional flexible data

    -- Audit fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for notification_logs
CREATE INDEX idx_notification_logs_payment_id ON notification_logs(payment_id) WHERE payment_id IS NOT NULL;
CREATE INDEX idx_notification_logs_merchant_id ON notification_logs(merchant_id);
CREATE INDEX idx_notification_logs_channel ON notification_logs(channel);
CREATE INDEX idx_notification_logs_status ON notification_logs(status);
CREATE INDEX idx_notification_logs_event_type ON notification_logs(event_type);
CREATE INDEX idx_notification_logs_created_at ON notification_logs(created_at DESC);
CREATE INDEX idx_notification_logs_retry ON notification_logs(next_retry_at, status)
    WHERE status = 'retrying' AND next_retry_at IS NOT NULL;

-- Composite index for delivery rate analysis
CREATE INDEX idx_notification_logs_channel_status_created ON notification_logs(channel, status, created_at DESC);

-- Table: merchant_notification_preferences - Merchant channel configuration
CREATE TABLE IF NOT EXISTS merchant_notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL, -- Reference to merchant
    channel VARCHAR(50) NOT NULL, -- 'email', 'telegram', 'zalo', 'webhook', 'speaker'

    -- Channel Enable/Disable
    enabled BOOLEAN DEFAULT TRUE,

    -- Event Subscriptions (which events trigger notifications on this channel)
    subscribed_events TEXT[] DEFAULT ARRAY['payment_confirmed', 'payment_expired', 'payment_failed'], -- Array of event types

    -- Channel-Specific Configuration (JSONB for flexibility)
    config JSONB DEFAULT '{}',
    /*
    Example config structures:

    Email: {
      "email_address": "merchant@example.com",
      "cc": ["accounting@example.com"],
      "include_qr_code": true
    }

    Telegram: {
      "chat_id": "123456789",
      "bot_username": "@payment_bot",
      "parse_mode": "HTML"
    }

    Zalo: {
      "phone_number": "+84901234567",
      "zns_template_id": "123456",
      "oa_id": "1234567890"
    }

    Webhook: {
      "url": "https://merchant.com/webhook",
      "secret": "webhook_secret_key",
      "timeout_seconds": 10,
      "retry_on_failure": true
    }

    Speaker/TTS: {
      "volume": 80,
      "voice": "vi-VN-Standard-A",
      "speed": 1.0,
      "device_id": "speaker_001"
    }
    */

    -- Rate Limiting
    rate_limit_per_hour INTEGER, -- NULL = no limit
    rate_limit_per_day INTEGER,  -- NULL = no limit

    -- Testing
    is_test_mode BOOLEAN DEFAULT FALSE, -- If true, notifications go to test endpoints
    test_config JSONB, -- Test-specific configuration

    -- Activity Tracking
    last_notification_at TIMESTAMP,
    total_notifications_sent INTEGER DEFAULT 0,

    -- Audit fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Unique constraint: one preference per merchant per channel
    CONSTRAINT unique_merchant_channel UNIQUE (merchant_id, channel)
);

-- Indexes for merchant_notification_preferences
CREATE UNIQUE INDEX idx_merchant_prefs_merchant_channel ON merchant_notification_preferences(merchant_id, channel);
CREATE INDEX idx_merchant_prefs_enabled ON merchant_notification_preferences(enabled) WHERE enabled = TRUE;
CREATE INDEX idx_merchant_prefs_channel ON merchant_notification_preferences(channel);

-- Trigger to update updated_at for notification_logs
CREATE OR REPLACE FUNCTION update_notification_logs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_notification_logs_updated_at
    BEFORE UPDATE ON notification_logs
    FOR EACH ROW
    EXECUTE FUNCTION update_notification_logs_updated_at();

-- Trigger to update updated_at for merchant_notification_preferences
CREATE OR REPLACE FUNCTION update_merchant_prefs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_merchant_prefs_updated_at
    BEFORE UPDATE ON merchant_notification_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_merchant_prefs_updated_at();

-- Check constraints
ALTER TABLE notification_logs ADD CONSTRAINT check_notification_status
    CHECK (status IN ('pending', 'sent', 'delivered', 'failed', 'retrying'));

ALTER TABLE notification_logs ADD CONSTRAINT check_notification_channel
    CHECK (channel IN ('email', 'telegram', 'zalo', 'webhook', 'speaker'));

ALTER TABLE notification_logs ADD CONSTRAINT check_retry_count
    CHECK (retry_count >= 0 AND retry_count <= max_retries);

ALTER TABLE merchant_notification_preferences ADD CONSTRAINT check_prefs_channel
    CHECK (channel IN ('email', 'telegram', 'zalo', 'webhook', 'speaker'));

-- View: Notification delivery statistics by channel
CREATE OR REPLACE VIEW notification_delivery_stats AS
SELECT
    channel,
    DATE(created_at) AS date,
    COUNT(*) AS total_attempts,
    COUNT(*) FILTER (WHERE status = 'delivered') AS delivered,
    COUNT(*) FILTER (WHERE status = 'failed') AS failed,
    COUNT(*) FILTER (WHERE status = 'pending' OR status = 'retrying') AS in_progress,
    ROUND(
        100.0 * COUNT(*) FILTER (WHERE status = 'delivered') / NULLIF(COUNT(*), 0),
        2
    ) AS delivery_rate_pct,
    AVG(EXTRACT(EPOCH FROM (delivered_at - created_at))) FILTER (WHERE delivered_at IS NOT NULL) AS avg_delivery_time_seconds
FROM notification_logs
GROUP BY channel, DATE(created_at)
ORDER BY date DESC, channel;

-- View: Notifications pending retry
CREATE OR REPLACE VIEW pending_retry_notifications AS
SELECT
    id,
    channel,
    event_type,
    merchant_id,
    payment_id,
    retry_count,
    max_retries,
    next_retry_at,
    error_message,
    created_at
FROM notification_logs
WHERE status = 'retrying'
    AND retry_count < max_retries
    AND next_retry_at <= CURRENT_TIMESTAMP
ORDER BY next_retry_at ASC;

-- Comments for documentation
COMMENT ON TABLE notification_logs IS 'Tracks all notification attempts across all channels (PRD v2.2 Omni-channel Notification Center)';
COMMENT ON TABLE merchant_notification_preferences IS 'Merchant-specific notification channel configuration and preferences';
COMMENT ON COLUMN notification_logs.channel IS 'Notification delivery channel: email, telegram, zalo, webhook, speaker';
COMMENT ON COLUMN notification_logs.status IS 'Current delivery status: pending → sent → delivered | failed → retrying';
COMMENT ON COLUMN merchant_notification_preferences.config IS 'Channel-specific configuration stored as flexible JSONB';
COMMENT ON COLUMN merchant_notification_preferences.subscribed_events IS 'Array of event types that trigger notifications on this channel';
COMMENT ON VIEW notification_delivery_stats IS 'Daily notification delivery success rate by channel';
COMMENT ON VIEW pending_retry_notifications IS 'Notifications that are due for retry (status=retrying and next_retry_at <= now)';
