-- Drop GIN indexes
DROP INDEX IF EXISTS idx_audit_logs_metadata_gin;
DROP INDEX IF EXISTS idx_merchants_metadata_gin;
DROP INDEX IF EXISTS idx_payments_metadata_gin;
DROP INDEX IF EXISTS idx_merchants_business_name_trgm;

-- Drop extensions (only if safe - check dependencies)
-- DROP EXTENSION IF EXISTS pg_trgm;
-- DROP EXTENSION IF EXISTS "uuid-ossp";

-- Drop functions
DROP FUNCTION IF EXISTS get_merchant_payment_stats(UUID, INT);
DROP FUNCTION IF EXISTS calculate_merchant_balance_from_ledger(UUID, VARCHAR);

-- Drop views
DROP VIEW IF EXISTS unmatched_blockchain_transactions;
DROP VIEW IF EXISTS system_statistics;
DROP VIEW IF EXISTS pending_payouts_for_review;
DROP VIEW IF EXISTS recent_payments_summary;
DROP VIEW IF EXISTS active_merchants_with_balance;
