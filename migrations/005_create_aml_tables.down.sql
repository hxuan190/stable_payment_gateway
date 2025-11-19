-- Migration Rollback: Drop AML (Anti-Money Laundering) Engine Tables
-- Version: 005
-- Description: Rollback script to remove all AML-related tables and functions
-- Created: 2025-11-19

-- WARNING: This will delete all AML data including:
-- - Customer risk scores
-- - Transaction monitoring records
-- - Alerts and cases
-- - Sanctions lists
-- - Wallet screening cache
-- - Audit logs
-- - Reports
--
-- This operation is IRREVERSIBLE. Ensure you have backups before proceeding.

-- ============================================================================
-- Drop Tables (in reverse order of dependencies)
-- ============================================================================

DROP TABLE IF EXISTS aml_audit_log CASCADE;
DROP TABLE IF EXISTS aml_reports CASCADE;
DROP TABLE IF EXISTS aml_rules CASCADE;
DROP TABLE IF EXISTS aml_wallet_screening CASCADE;
DROP TABLE IF EXISTS aml_sanctions_list CASCADE;
DROP TABLE IF EXISTS aml_cases CASCADE;
DROP TABLE IF EXISTS aml_alerts CASCADE;
DROP TABLE IF EXISTS aml_transaction_monitoring CASCADE;
DROP TABLE IF EXISTS aml_customer_risk_scores CASCADE;

-- ============================================================================
-- Drop Sequences
-- ============================================================================

DROP SEQUENCE IF EXISTS aml_case_number_seq CASCADE;

-- ============================================================================
-- Drop Functions (if created only for AML)
-- ============================================================================

-- Note: update_updated_at_column() function may be used by other tables
-- Only drop if you're certain it's not needed elsewhere
-- DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;

-- ============================================================================
-- Rollback Complete
-- ============================================================================

-- Optional: Add comment to track rollback
COMMENT ON SCHEMA public IS 'AML Engine tables dropped - Migration 005 rolled back';
