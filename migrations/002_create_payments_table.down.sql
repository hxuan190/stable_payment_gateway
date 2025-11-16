-- Drop trigger
DROP TRIGGER IF EXISTS update_payments_updated_at ON payments;

-- Drop indexes
DROP INDEX IF EXISTS idx_payments_merchant_status;
DROP INDEX IF EXISTS idx_payments_expires_at;
DROP INDEX IF EXISTS idx_payments_created_at;
DROP INDEX IF EXISTS idx_payments_order_id;
DROP INDEX IF EXISTS idx_payments_payment_reference;
DROP INDEX IF EXISTS idx_payments_tx_hash;
DROP INDEX IF EXISTS idx_payments_status;
DROP INDEX IF EXISTS idx_payments_merchant_id;

-- Drop table
DROP TABLE IF EXISTS payments;
