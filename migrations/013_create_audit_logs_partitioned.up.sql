-- Create partitioned audit logs table (Phase 1: Shadow Table Approach)
-- This creates audit_logs_v2 alongside existing audit_logs for safe migration
-- See: docs/SAFE_AUDIT_LOGS_MIGRATION.md for full migration strategy

-- Create partitioned table with new name (not replacing existing table yet)
CREATE TABLE IF NOT EXISTS audit_logs_v2 (
    id UUID NOT NULL,

    -- Who performed the action
    actor_type VARCHAR(50) NOT NULL, -- system, merchant, admin, ops, api, worker, listener
    actor_id UUID, -- merchant_id, admin_id, NULL for system
    actor_email VARCHAR(255),
    actor_ip_address VARCHAR(45), -- IPv4 or IPv6

    -- What action was performed
    action VARCHAR(100) NOT NULL, -- payment_created, payment_completed, kyc_approved, payout_requested, etc.
    action_category VARCHAR(50) NOT NULL, -- payment, payout, kyc, authentication, admin, system, blockchain, webhook, security

    -- What resource was affected
    resource_type VARCHAR(50) NOT NULL, -- payment, payout, merchant, ledger_entry, etc.
    resource_id UUID NOT NULL,

    -- Action result
    status VARCHAR(20) NOT NULL, -- success, failed, error
    error_message TEXT,

    -- Request details
    http_method VARCHAR(10),
    http_path TEXT,
    http_status_code INT,
    user_agent TEXT,

    -- Before and after state (for updates)
    old_values JSONB,
    new_values JSONB,

    -- Additional context
    metadata JSONB,
    description TEXT,

    -- Correlation for request tracing
    request_id UUID,
    correlation_id UUID,

    -- Timestamp - immutable, never updated (used as partition key)
    created_at TIMESTAMP NOT NULL DEFAULT NOW()

    -- No updated_at or deleted_at - audit logs are immutable
) PARTITION BY RANGE (created_at);

-- Create partitions for 5 years (2025-2030) for compliance retention
CREATE TABLE audit_logs_v2_2025 PARTITION OF audit_logs_v2
    FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');

CREATE TABLE audit_logs_v2_2026 PARTITION OF audit_logs_v2
    FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');

CREATE TABLE audit_logs_v2_2027 PARTITION OF audit_logs_v2
    FOR VALUES FROM ('2027-01-01') TO ('2028-01-01');

CREATE TABLE audit_logs_v2_2028 PARTITION OF audit_logs_v2
    FOR VALUES FROM ('2028-01-01') TO ('2029-01-01');

CREATE TABLE audit_logs_v2_2029 PARTITION OF audit_logs_v2
    FOR VALUES FROM ('2029-01-01') TO ('2030-01-01');

CREATE TABLE audit_logs_v2_2030 PARTITION OF audit_logs_v2
    FOR VALUES FROM ('2030-01-01') TO ('2031-01-01');

-- Create indexes (on partitioned table, applies to all partitions)
-- Indexes are automatically created on each partition
CREATE INDEX idx_audit_logs_v2_actor_id ON audit_logs_v2(actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_audit_logs_v2_actor_type ON audit_logs_v2(actor_type);
CREATE INDEX idx_audit_logs_v2_action ON audit_logs_v2(action);
CREATE INDEX idx_audit_logs_v2_action_category ON audit_logs_v2(action_category);
CREATE INDEX idx_audit_logs_v2_resource ON audit_logs_v2(resource_type, resource_id);
CREATE INDEX idx_audit_logs_v2_status ON audit_logs_v2(status);
CREATE INDEX idx_audit_logs_v2_created_at ON audit_logs_v2(created_at DESC);
CREATE INDEX idx_audit_logs_v2_request_id ON audit_logs_v2(request_id) WHERE request_id IS NOT NULL;
CREATE INDEX idx_audit_logs_v2_actor_ip ON audit_logs_v2(actor_ip_address) WHERE actor_ip_address IS NOT NULL;

-- Composite indexes for common queries
CREATE INDEX idx_audit_logs_v2_actor_action ON audit_logs_v2(actor_type, action, created_at DESC);
CREATE INDEX idx_audit_logs_v2_resource_action ON audit_logs_v2(resource_type, resource_id, action, created_at DESC);

-- Add constraints (same as original table)
ALTER TABLE audit_logs_v2 ADD CONSTRAINT check_audit_v2_actor_type
    CHECK (actor_type IN ('system', 'merchant', 'admin', 'ops', 'api', 'worker', 'listener'));

ALTER TABLE audit_logs_v2 ADD CONSTRAINT check_audit_v2_action_category
    CHECK (action_category IN ('payment', 'payout', 'kyc', 'authentication', 'admin', 'system', 'blockchain', 'webhook', 'security'));

ALTER TABLE audit_logs_v2 ADD CONSTRAINT check_audit_v2_status
    CHECK (status IN ('success', 'failed', 'error'));

-- Add comments for documentation
COMMENT ON TABLE audit_logs_v2 IS 'Partitioned audit logs (v2) - replaces audit_logs after migration. See docs/SAFE_AUDIT_LOGS_MIGRATION.md';
COMMENT ON COLUMN audit_logs_v2.actor_type IS 'Type of actor: system, merchant, admin, ops, api, worker, listener';
COMMENT ON COLUMN audit_logs_v2.actor_id IS 'ID of the actor (merchant_id, admin_id, etc.)';
COMMENT ON COLUMN audit_logs_v2.action IS 'Specific action performed (payment_created, kyc_approved, etc.)';
COMMENT ON COLUMN audit_logs_v2.action_category IS 'Category of action: payment, payout, kyc, authentication, admin, system, blockchain, webhook, security';
COMMENT ON COLUMN audit_logs_v2.resource_type IS 'Type of resource affected: payment, payout, merchant, ledger_entry, etc.';
COMMENT ON COLUMN audit_logs_v2.resource_id IS 'ID of the affected resource';
COMMENT ON COLUMN audit_logs_v2.old_values IS 'Resource state before the action (for updates)';
COMMENT ON COLUMN audit_logs_v2.new_values IS 'Resource state after the action (for updates)';
COMMENT ON COLUMN audit_logs_v2.request_id IS 'Unique request identifier for tracing';
COMMENT ON COLUMN audit_logs_v2.correlation_id IS 'Correlation ID for linking related operations';
COMMENT ON COLUMN audit_logs_v2.created_at IS 'Timestamp of audit log (partition key - IMMUTABLE)';

-- Note: This is Phase 1 of the migration. The old audit_logs table remains active.
-- Next phases:
-- Phase 2: Implement dual-write in application code (write to both tables)
-- Phase 3: Backfill historical data from audit_logs to audit_logs_v2
-- Phase 4: Verify data integrity
-- Phase 5: Switch reads to audit_logs_v2
-- Phase 6: Rename audit_logs_v2 to audit_logs (atomic swap)
-- Phase 7: Drop old table after 1 week safety period
