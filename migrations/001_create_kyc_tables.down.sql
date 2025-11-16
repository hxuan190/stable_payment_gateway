-- Migration: Rollback KYC tables

-- Drop audit logs first (no foreign key dependencies)
DROP TABLE IF EXISTS kyc_audit_logs;

-- Drop review actions
DROP TABLE IF EXISTS kyc_review_actions;

-- Drop risk assessments
DROP TABLE IF EXISTS kyc_risk_assessments;

-- Drop verification results
DROP TABLE IF EXISTS kyc_verification_results;

-- Drop documents
DROP TABLE IF EXISTS kyc_documents;

-- Drop submissions
DROP TABLE IF EXISTS kyc_submissions;

-- Remove columns from merchants table
ALTER TABLE merchants
    DROP COLUMN IF EXISTS merchant_type,
    DROP COLUMN IF EXISTS full_name,
    DROP COLUMN IF EXISTS phone,
    DROP COLUMN IF EXISTS bank_account_number,
    DROP COLUMN IF EXISTS bank_account_name,
    DROP COLUMN IF EXISTS bank_name,
    DROP COLUMN IF EXISTS daily_limit_vnd,
    DROP COLUMN IF EXISTS monthly_limit_vnd;

-- Drop enum types
DROP TYPE IF EXISTS risk_level_enum;
DROP TYPE IF EXISTS verification_status_enum;
DROP TYPE IF EXISTS kyc_document_type_enum;
DROP TYPE IF EXISTS kyc_status_enum;
DROP TYPE IF EXISTS merchant_type_enum;
