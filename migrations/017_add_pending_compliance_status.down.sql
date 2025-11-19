-- Rollback Migration 017: Remove pending_compliance status

-- Drop indexes
DROP INDEX IF EXISTS idx_payments_compliance_expired;
DROP INDEX IF EXISTS idx_payments_pending_compliance;

-- Remove compliance columns
ALTER TABLE payments
DROP COLUMN IF EXISTS compliance_submitted_at;

ALTER TABLE payments
DROP COLUMN IF EXISTS compliance_deadline;

-- Restore original status constraint
ALTER TABLE payments
DROP CONSTRAINT IF EXISTS payments_status_check;

ALTER TABLE payments
ADD CONSTRAINT payments_status_check
CHECK (status IN ('created', 'pending', 'confirming', 'completed', 'expired', 'failed'));
