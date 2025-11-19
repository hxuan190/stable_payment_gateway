-- Rollback Migration 018: Remove OTC Partner Liquidity Monitoring

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_update_otc_balance_flags ON otc_partner_liquidity;
DROP TRIGGER IF EXISTS trigger_otc_partner_liquidity_updated_at ON otc_partner_liquidity;

-- Drop functions
DROP FUNCTION IF EXISTS update_otc_balance_flags();
DROP FUNCTION IF EXISTS update_otc_partner_liquidity_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_otc_alerts_created;
DROP INDEX IF EXISTS idx_otc_alerts_unresolved;
DROP INDEX IF EXISTS idx_otc_alerts_partner;

DROP INDEX IF EXISTS idx_otc_liquidity_history_recorded;
DROP INDEX IF EXISTS idx_otc_liquidity_history_partner;

DROP INDEX IF EXISTS idx_otc_partner_liquidity_updated;
DROP INDEX IF EXISTS idx_otc_partner_liquidity_alerts;
DROP INDEX IF EXISTS idx_otc_partner_liquidity_balance;
DROP INDEX IF EXISTS idx_otc_partner_liquidity_status;

-- Drop tables
DROP TABLE IF EXISTS otc_liquidity_alerts;
DROP TABLE IF EXISTS otc_partner_liquidity_history;
DROP TABLE IF EXISTS otc_partner_liquidity;
