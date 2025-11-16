-- Create audit_logs table
-- This table stores all critical operations for compliance and debugging
-- IMPORTANT: This table is append-only (no updates or deletes)
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Who performed the action
    actor_type VARCHAR(50) NOT NULL, -- system, merchant, admin, ops
    actor_id UUID, -- merchant_id, admin_id, NULL for system
    actor_email VARCHAR(255),
    actor_ip_address VARCHAR(45), -- IPv4 or IPv6

    -- What action was performed
    action VARCHAR(100) NOT NULL, -- payment_created, payment_completed, kyc_approved, payout_requested, etc.
    action_category VARCHAR(50) NOT NULL, -- payment, payout, kyc, authentication, admin, system

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

    -- Timestamp - immutable, never updated
    created_at TIMESTAMP NOT NULL DEFAULT NOW()

    -- No updated_at or deleted_at - audit logs are immutable
);

-- Create indexes for efficient querying
CREATE INDEX idx_audit_logs_actor_id ON audit_logs(actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_audit_logs_actor_type ON audit_logs(actor_type);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_action_category ON audit_logs(action_category);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_status ON audit_logs(status);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_request_id ON audit_logs(request_id) WHERE request_id IS NOT NULL;
CREATE INDEX idx_audit_logs_actor_ip ON audit_logs(actor_ip_address) WHERE actor_ip_address IS NOT NULL;

-- Composite indexes for common queries
CREATE INDEX idx_audit_logs_actor_action ON audit_logs(actor_type, action, created_at DESC);
CREATE INDEX idx_audit_logs_resource_action ON audit_logs(resource_type, resource_id, action, created_at DESC);

-- Add constraints
ALTER TABLE audit_logs ADD CONSTRAINT check_audit_actor_type
    CHECK (actor_type IN ('system', 'merchant', 'admin', 'ops', 'api', 'worker', 'listener'));

ALTER TABLE audit_logs ADD CONSTRAINT check_audit_action_category
    CHECK (action_category IN ('payment', 'payout', 'kyc', 'authentication', 'admin', 'system', 'blockchain', 'webhook', 'security'));

ALTER TABLE audit_logs ADD CONSTRAINT check_audit_status
    CHECK (status IN ('success', 'failed', 'error'));

-- Add comments for documentation
COMMENT ON TABLE audit_logs IS 'Comprehensive audit trail for all critical operations (append-only, immutable)';
COMMENT ON COLUMN audit_logs.actor_type IS 'Type of actor: system, merchant, admin, ops, api, worker, listener';
COMMENT ON COLUMN audit_logs.actor_id IS 'ID of the actor (merchant_id, admin_id, etc.)';
COMMENT ON COLUMN audit_logs.action IS 'Specific action performed (payment_created, kyc_approved, etc.)';
COMMENT ON COLUMN audit_logs.action_category IS 'Category of action: payment, payout, kyc, authentication, admin, system, blockchain, webhook, security';
COMMENT ON COLUMN audit_logs.resource_type IS 'Type of resource affected: payment, payout, merchant, ledger_entry, etc.';
COMMENT ON COLUMN audit_logs.resource_id IS 'ID of the affected resource';
COMMENT ON COLUMN audit_logs.old_values IS 'Resource state before the action (for updates)';
COMMENT ON COLUMN audit_logs.new_values IS 'Resource state after the action (for updates)';
COMMENT ON COLUMN audit_logs.request_id IS 'Unique request identifier for tracing';
COMMENT ON COLUMN audit_logs.correlation_id IS 'Correlation ID for linking related operations';

-- Create partitioning by month for better performance (optional, can be added later)
-- This helps manage the growing size of audit logs over time
COMMENT ON TABLE audit_logs IS 'Comprehensive audit trail for all critical operations (append-only, immutable). Consider partitioning by created_at for large-scale deployments.';
