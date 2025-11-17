-- Drop indexes
DROP INDEX IF EXISTS idx_wallet_balance_snapshots_chain_wallet_snapshot;
DROP INDEX IF EXISTS idx_wallet_balance_snapshots_created_at;
DROP INDEX IF EXISTS idx_wallet_balance_snapshots_alert_status;
DROP INDEX IF EXISTS idx_wallet_balance_snapshots_snapshot_at;
DROP INDEX IF EXISTS idx_wallet_balance_snapshots_wallet_address;
DROP INDEX IF EXISTS idx_wallet_balance_snapshots_chain_network;

-- Drop table
DROP TABLE IF EXISTS wallet_balance_snapshots;
