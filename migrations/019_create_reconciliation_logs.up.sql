-- Migration: Create reconciliation_logs table for daily solvency audits
-- Purpose: Track daily reconciliation checks to ensure Assets >= Liabilities
-- Formula: Assets = Crypto Holdings + VND Pool + Fee Revenue - Expenses
--          Liabilities = Sum of all merchant balances
--          Difference = Assets - Liabilities

-- Create reconciliation_logs table
CREATE TABLE IF NOT EXISTS reconciliation_logs (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Financial totals (in VND)
    total_assets_vnd DECIMAL(20, 2) NOT NULL DEFAULT 0,
    total_liabilities_vnd DECIMAL(20, 2) NOT NULL DEFAULT 0,
    difference_vnd DECIMAL(20, 2) NOT NULL DEFAULT 0, -- Assets - Liabilities

    -- Detailed asset breakdown (JSONB for flexibility)
    -- Example: {"crypto_pool_vnd": "1000000", "vnd_pool": "500000", "fee_revenue": "50000"}
    asset_breakdown JSONB,

    -- Status of reconciliation
    -- 'balanced': Assets == Liabilities (within tolerance)
    -- 'surplus': Assets > Liabilities (good - accumulated fees)
    -- 'deficit': Assets < Liabilities (CRITICAL - system insolvent)
    -- 'error': Reconciliation failed to complete
    -- 'in_progress': Currently running
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress',
    CONSTRAINT check_recon_status CHECK (status IN ('balanced', 'surplus', 'deficit', 'error', 'in_progress')),

    -- Alert flag - set to true if critical alert triggered (deficit detected)
    alert_triggered BOOLEAN NOT NULL DEFAULT FALSE,

    -- Error message if reconciliation failed
    error_message TEXT,

    -- Timestamps
    reconciled_at TIMESTAMP NOT NULL DEFAULT NOW(), -- When reconciliation was performed
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_recon_logs_reconciled_at ON reconciliation_logs(reconciled_at DESC);
CREATE INDEX idx_recon_logs_status ON reconciliation_logs(status);
CREATE INDEX idx_recon_logs_alert_triggered ON reconciliation_logs(alert_triggered) WHERE alert_triggered = TRUE;
CREATE INDEX idx_recon_logs_deficit ON reconciliation_logs(status, reconciled_at DESC) WHERE status = 'deficit';

-- Add comments for documentation
COMMENT ON TABLE reconciliation_logs IS 'Daily reconciliation audit logs to verify system solvency (Assets >= Liabilities)';
COMMENT ON COLUMN reconciliation_logs.total_assets_vnd IS 'Total platform assets: Crypto holdings + VND pool + Revenue - Expenses (in VND)';
COMMENT ON COLUMN reconciliation_logs.total_liabilities_vnd IS 'Total liabilities: Sum of all merchant balances (available + pending + reserved)';
COMMENT ON COLUMN reconciliation_logs.difference_vnd IS 'Assets - Liabilities. Negative = deficit (CRITICAL), Positive = surplus (good)';
COMMENT ON COLUMN reconciliation_logs.asset_breakdown IS 'JSONB breakdown of all asset sources for audit trail';
COMMENT ON COLUMN reconciliation_logs.status IS 'Reconciliation result: balanced, surplus, deficit, error, in_progress';
COMMENT ON COLUMN reconciliation_logs.alert_triggered IS 'True if critical alert sent to ops team (deficit detected)';

-- Grant permissions (adjust based on your user roles)
-- GRANT SELECT ON reconciliation_logs TO readonly_user;
-- GRANT INSERT, SELECT ON reconciliation_logs TO app_user;
