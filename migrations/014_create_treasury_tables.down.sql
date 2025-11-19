-- Rollback: Drop treasury tables

-- Drop views
DROP VIEW IF EXISTS treasury_balance_summary CASCADE;
DROP VIEW IF EXISTS treasury_operation_summary CASCADE;
DROP VIEW IF EXISTS hot_wallets_needing_sweep CASCADE;

-- Drop triggers and functions
DROP TRIGGER IF EXISTS trigger_set_treasury_wallet_address_hash ON treasury_wallets;
DROP TRIGGER IF EXISTS trigger_update_treasury_operations_updated_at ON treasury_operations;
DROP TRIGGER IF EXISTS trigger_update_treasury_wallets_updated_at ON treasury_wallets;
DROP FUNCTION IF EXISTS set_treasury_wallet_address_hash();
DROP FUNCTION IF EXISTS update_treasury_operations_updated_at();
DROP FUNCTION IF EXISTS update_treasury_wallets_updated_at();

-- Drop tables (treasury_operations first due to foreign key)
DROP TABLE IF EXISTS treasury_operations CASCADE;
DROP TABLE IF EXISTS treasury_wallets CASCADE;
