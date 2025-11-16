-- Drop trigger
DROP TRIGGER IF EXISTS update_merchants_updated_at ON merchants;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_merchants_status;
DROP INDEX IF EXISTS idx_merchants_kyc_status;
DROP INDEX IF EXISTS idx_merchants_api_key;
DROP INDEX IF EXISTS idx_merchants_email;

-- Drop table
DROP TABLE IF EXISTS merchants;
