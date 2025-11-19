-- Rollback Migration 016: Remove 4-Layer Verification fields

-- Drop trigger and functions
DROP TRIGGER IF EXISTS trigger_update_verification_tier ON travel_rule_data;
DROP FUNCTION IF EXISTS update_verification_tier();
DROP FUNCTION IF EXISTS calculate_verification_tier(UUID);

-- Drop view
DROP VIEW IF EXISTS v_travel_rule_verification_status;

-- Drop indexes
DROP INDEX IF EXISTS idx_travel_rule_signature_verified;
DROP INDEX IF EXISTS idx_travel_rule_sanctions_status;
DROP INDEX IF EXISTS idx_travel_rule_wallet_sanctioned;
DROP INDEX IF EXISTS idx_travel_rule_ekyc_status;
DROP INDEX IF EXISTS idx_travel_rule_geo_mismatch;
DROP INDEX IF EXISTS idx_travel_rule_verification_tier;

-- Remove constraints
ALTER TABLE travel_rule_data DROP CONSTRAINT IF EXISTS check_sanctions_status;
ALTER TABLE travel_rule_data DROP CONSTRAINT IF EXISTS check_sanctions_result;
ALTER TABLE travel_rule_data DROP CONSTRAINT IF EXISTS check_ekyc_status;
ALTER TABLE travel_rule_data DROP CONSTRAINT IF EXISTS check_ekyc_document_type;
ALTER TABLE travel_rule_data DROP CONSTRAINT IF EXISTS check_verification_tier;

-- Remove columns (in reverse order of addition)
ALTER TABLE travel_rule_data
DROP COLUMN IF EXISTS verification_notes,
DROP COLUMN IF EXISTS verification_completed_at,
DROP COLUMN IF EXISTS verification_tier,
DROP COLUMN IF EXISTS ekyc_reference,
DROP COLUMN IF EXISTS ekyc_provider,
DROP COLUMN IF EXISTS ekyc_verified_at,
DROP COLUMN IF EXISTS ekyc_document_id,
DROP COLUMN IF EXISTS ekyc_document_type,
DROP COLUMN IF EXISTS ekyc_status,
DROP COLUMN IF EXISTS ekyc_required,
DROP COLUMN IF EXISTS anomaly_flags,
DROP COLUMN IF EXISTS anomaly_score,
DROP COLUMN IF EXISTS geo_mismatch_detected,
DROP COLUMN IF EXISTS payer_device_fingerprint,
DROP COLUMN IF EXISTS payer_user_agent,
DROP COLUMN IF EXISTS payer_ip_country,
DROP COLUMN IF EXISTS payer_ip_address,
DROP COLUMN IF EXISTS wallet_sanctioned,
DROP COLUMN IF EXISTS chainalysis_risk_score,
DROP COLUMN IF EXISTS sanctions_hit_details,
DROP COLUMN IF EXISTS sanctions_check_result,
DROP COLUMN IF EXISTS sanctions_check_status,
DROP COLUMN IF EXISTS signature_verified_at,
DROP COLUMN IF EXISTS signature_verified,
DROP COLUMN IF EXISTS signature_message,
DROP COLUMN IF EXISTS payer_signature;
