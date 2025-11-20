-- Rollback: Drop reconciliation_logs table

-- Drop indexes first
DROP INDEX IF EXISTS idx_recon_logs_deficit;
DROP INDEX IF EXISTS idx_recon_logs_alert_triggered;
DROP INDEX IF EXISTS idx_recon_logs_status;
DROP INDEX IF EXISTS idx_recon_logs_reconciled_at;

-- Drop table
DROP TABLE IF EXISTS reconciliation_logs;
