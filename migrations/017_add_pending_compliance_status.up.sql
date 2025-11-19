-- Migration 017: Add pending_compliance status to payments table
-- This status is used for payments waiting for FATF Travel Rule data submission

-- Add pending_compliance to payment status enum
ALTER TABLE payments
DROP CONSTRAINT IF EXISTS payments_status_check;

ALTER TABLE payments
ADD CONSTRAINT payments_status_check
CHECK (status IN ('created', 'pending', 'pending_compliance', 'confirming', 'completed', 'expired', 'failed'));

-- Add compliance deadline field to track when compliance data must be submitted
ALTER TABLE payments
ADD COLUMN IF NOT EXISTS compliance_deadline TIMESTAMP;

-- Add compliance submitted timestamp
ALTER TABLE payments
ADD COLUMN IF NOT EXISTS compliance_submitted_at TIMESTAMP;

-- Create index for querying payments pending compliance
CREATE INDEX IF NOT EXISTS idx_payments_pending_compliance
ON payments(status, compliance_deadline)
WHERE status = 'pending_compliance';

-- Create index for compliance expiry cronjob queries
CREATE INDEX IF NOT EXISTS idx_payments_compliance_expired
ON payments(compliance_deadline)
WHERE status = 'pending_compliance' AND compliance_deadline IS NOT NULL;

COMMENT ON COLUMN payments.compliance_deadline IS 'Deadline for submitting FATF Travel Rule compliance data (24 hours from payment creation for high-value transactions)';
COMMENT ON COLUMN payments.compliance_submitted_at IS 'Timestamp when Travel Rule compliance data was successfully submitted';
