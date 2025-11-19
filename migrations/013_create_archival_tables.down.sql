-- Rollback: Drop archival tables

-- Drop views
DROP VIEW IF EXISTS daily_hash_chain_summary CASCADE;
DROP VIEW IF EXISTS pending_archive_restores CASCADE;
DROP VIEW IF EXISTS archive_storage_stats CASCADE;

-- Drop functions
DROP FUNCTION IF EXISTS verify_hash_chain(VARCHAR, DATE, DATE);
DROP FUNCTION IF EXISTS compute_data_hash(JSONB);

-- Drop triggers and functions
DROP TRIGGER IF EXISTS trigger_update_transaction_hashes_updated_at ON transaction_hashes;
DROP TRIGGER IF EXISTS trigger_update_archived_records_updated_at ON archived_records;
DROP FUNCTION IF EXISTS update_transaction_hashes_updated_at();
DROP FUNCTION IF EXISTS update_archived_records_updated_at();

-- Drop tables
DROP TABLE IF EXISTS transaction_hashes CASCADE;
DROP TABLE IF EXISTS archived_records CASCADE;
