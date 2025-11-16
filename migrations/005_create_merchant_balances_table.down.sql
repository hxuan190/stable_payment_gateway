-- Drop triggers
DROP TRIGGER IF EXISTS create_merchant_balance_on_merchant_insert ON merchants;
DROP TRIGGER IF EXISTS update_merchant_balances_updated_at ON merchant_balances;

-- Drop functions
DROP FUNCTION IF EXISTS create_merchant_balance();

-- Drop indexes
DROP INDEX IF EXISTS idx_merchant_balances_merchant_id;

-- Drop table
DROP TABLE IF EXISTS merchant_balances;
