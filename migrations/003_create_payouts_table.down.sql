-- Drop trigger
DROP TRIGGER IF EXISTS update_payouts_updated_at ON payouts;

-- Drop indexes
DROP INDEX IF EXISTS idx_payouts_approved_at;
DROP INDEX IF EXISTS idx_payouts_created_at;
DROP INDEX IF EXISTS idx_payouts_merchant_status;
DROP INDEX IF EXISTS idx_payouts_status;
DROP INDEX IF EXISTS idx_payouts_merchant_id;

-- Drop table
DROP TABLE IF EXISTS payouts;
