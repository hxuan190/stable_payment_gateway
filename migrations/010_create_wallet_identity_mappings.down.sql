-- Rollback: Drop wallet_identity_mappings table

-- Drop views
DROP VIEW IF EXISTS wallet_activity_summary CASCADE;
DROP VIEW IF EXISTS active_wallet_mappings CASCADE;

-- Drop triggers and functions
DROP TRIGGER IF EXISTS trigger_set_wallet_address_hash ON wallet_identity_mappings;
DROP TRIGGER IF EXISTS trigger_update_wallet_mappings_updated_at ON wallet_identity_mappings;
DROP FUNCTION IF EXISTS set_wallet_address_hash();
DROP FUNCTION IF EXISTS compute_wallet_hash(TEXT);
DROP FUNCTION IF EXISTS update_wallet_mappings_updated_at();

-- Drop table
DROP TABLE IF EXISTS wallet_identity_mappings CASCADE;
