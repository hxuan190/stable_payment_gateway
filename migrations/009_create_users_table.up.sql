-- Migration: Create users table for end-user identity (PRD v2.2)
-- Purpose: Store end-user information for KYC and wallet identity mapping

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Basic Information
    full_name VARCHAR(255) NOT NULL,
    date_of_birth DATE,
    nationality VARCHAR(3), -- ISO 3166-1 alpha-3 country code
    email VARCHAR(255),
    phone VARCHAR(50),

    -- KYC Information
    kyc_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    kyc_provider VARCHAR(50) DEFAULT 'sumsub', -- 'sumsub', 'onfido', 'manual'
    kyc_applicant_id VARCHAR(255), -- External KYC provider applicant ID
    kyc_verified_at TIMESTAMP,
    kyc_rejection_reason TEXT,
    face_liveness_passed BOOLEAN DEFAULT FALSE,

    -- Identity Documents
    document_type VARCHAR(50), -- 'passport', 'national_id', 'driving_license'
    document_number VARCHAR(100),
    document_country VARCHAR(3), -- ISO 3166-1 alpha-3
    document_expiry_date DATE,

    -- Risk Assessment
    risk_level VARCHAR(20) DEFAULT 'low', -- 'low', 'medium', 'high'
    aml_checked_at TIMESTAMP,
    is_pep BOOLEAN DEFAULT FALSE, -- Politically Exposed Person
    is_sanctioned BOOLEAN DEFAULT FALSE,

    -- Activity Tracking
    first_payment_at TIMESTAMP,
    last_payment_at TIMESTAMP,
    total_payment_count INTEGER DEFAULT 0,
    total_payment_volume_usd DECIMAL(20, 2) DEFAULT 0,

    -- Metadata
    metadata JSONB DEFAULT '{}', -- Additional flexible data
    notes TEXT, -- Admin notes

    -- Audit fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(100), -- Admin user who created the record
    updated_by VARCHAR(100)  -- Admin user who last updated the record
);

-- Indexes for performance
CREATE INDEX idx_users_kyc_status ON users(kyc_status);
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_phone ON users(phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_users_kyc_applicant_id ON users(kyc_applicant_id) WHERE kyc_applicant_id IS NOT NULL;
CREATE INDEX idx_users_document_number ON users(document_number) WHERE document_number IS NOT NULL;
CREATE INDEX idx_users_risk_level ON users(risk_level);
CREATE INDEX idx_users_created_at ON users(created_at DESC);
CREATE INDEX idx_users_last_payment_at ON users(last_payment_at DESC);

-- Trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_users_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_users_updated_at();

-- Add check constraints
ALTER TABLE users ADD CONSTRAINT check_users_kyc_status
    CHECK (kyc_status IN ('pending', 'in_progress', 'approved', 'rejected', 'expired'));

ALTER TABLE users ADD CONSTRAINT check_users_risk_level
    CHECK (risk_level IN ('low', 'medium', 'high', 'critical'));

ALTER TABLE users ADD CONSTRAINT check_users_document_type
    CHECK (document_type IN ('passport', 'national_id', 'driving_license', 'other') OR document_type IS NULL);

-- Comments for documentation
COMMENT ON TABLE users IS 'End-user identity table for KYC and wallet identity mapping (PRD v2.2)';
COMMENT ON COLUMN users.kyc_status IS 'Current KYC verification status';
COMMENT ON COLUMN users.kyc_applicant_id IS 'External KYC provider (Sumsub) applicant identifier';
COMMENT ON COLUMN users.face_liveness_passed IS 'Whether user passed face liveness detection (anti-spoofing)';
COMMENT ON COLUMN users.is_pep IS 'Politically Exposed Person flag for enhanced due diligence';
COMMENT ON COLUMN users.total_payment_volume_usd IS 'Cumulative payment volume across all wallets for AML monitoring';
