-- Rollback Migration 015: Remove FATF Travel Rule enhancements

-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_travel_rule_updated_at ON travel_rule_data;
DROP FUNCTION IF EXISTS update_travel_rule_updated_at();
DROP FUNCTION IF EXISTS check_travel_rule_compliance(DECIMAL, VARCHAR, VARCHAR, CHAR, CHAR);

-- Drop view
DROP VIEW IF EXISTS v_travel_rule_regulatory_report;

-- Drop indexes
DROP INDEX IF EXISTS idx_travel_rule_merchant_wallet;
DROP INDEX IF EXISTS idx_travel_rule_risk_level;
DROP INDEX IF EXISTS idx_travel_rule_screening_status;
DROP INDEX IF EXISTS idx_travel_rule_reported;
DROP INDEX IF EXISTS idx_travel_rule_transaction_purpose;

-- Remove constraints
ALTER TABLE travel_rule_data DROP CONSTRAINT IF EXISTS check_travel_rule_risk_level;
ALTER TABLE travel_rule_data DROP CONSTRAINT IF EXISTS check_travel_rule_screening_status;
ALTER TABLE travel_rule_data DROP CONSTRAINT IF EXISTS check_travel_rule_transaction_purpose;
ALTER TABLE travel_rule_data DROP CONSTRAINT IF EXISTS check_travel_rule_retention_policy;

-- Remove columns (in reverse order of addition)
ALTER TABLE travel_rule_data
DROP COLUMN IF EXISTS updated_at,
DROP COLUMN IF EXISTS archived_at,
DROP COLUMN IF EXISTS retention_policy,
DROP COLUMN IF EXISTS report_reference,
DROP COLUMN IF EXISTS reported_at,
DROP COLUMN IF EXISTS reported_to_authority,
DROP COLUMN IF EXISTS screening_completed_at,
DROP COLUMN IF EXISTS screening_status,
DROP COLUMN IF EXISTS risk_score,
DROP COLUMN IF EXISTS risk_level,
DROP COLUMN IF EXISTS merchant_address,
DROP COLUMN IF EXISTS payer_address,
DROP COLUMN IF EXISTS payer_date_of_birth,
DROP COLUMN IF EXISTS transaction_purpose,
DROP COLUMN IF EXISTS merchant_business_registration,
DROP COLUMN IF EXISTS merchant_id_document,
DROP COLUMN IF EXISTS merchant_wallet_address;
