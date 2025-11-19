-- Migration 016: Add 4-Layer Verification for Unhosted Wallet Travel Rule
-- Reference: Risk-Based Approach for Unhosted Wallet Verification

-- Add Layer 1: Proof of Ownership (Signature Verification)
ALTER TABLE travel_rule_data
ADD COLUMN IF NOT EXISTS payer_signature TEXT,
ADD COLUMN IF NOT EXISTS signature_message TEXT,
ADD COLUMN IF NOT EXISTS signature_verified BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS signature_verified_at TIMESTAMP;

-- Add Layer 2: Sanctions Screening
ALTER TABLE travel_rule_data
ADD COLUMN IF NOT EXISTS sanctions_check_status VARCHAR(50) DEFAULT 'pending',
ADD COLUMN IF NOT EXISTS sanctions_check_result VARCHAR(50),
ADD COLUMN IF NOT EXISTS sanctions_hit_details JSONB,
ADD COLUMN IF NOT EXISTS chainalysis_risk_score DECIMAL(5, 2),
ADD COLUMN IF NOT EXISTS wallet_sanctioned BOOLEAN DEFAULT FALSE;

-- Add Layer 3: Passive Signals (Anomaly Detection)
ALTER TABLE travel_rule_data
ADD COLUMN IF NOT EXISTS payer_ip_address VARCHAR(45),
ADD COLUMN IF NOT EXISTS payer_ip_country VARCHAR(2),
ADD COLUMN IF NOT EXISTS payer_user_agent TEXT,
ADD COLUMN IF NOT EXISTS payer_device_fingerprint VARCHAR(255),
ADD COLUMN IF NOT EXISTS geo_mismatch_detected BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS anomaly_score DECIMAL(5, 2),
ADD COLUMN IF NOT EXISTS anomaly_flags JSONB;

-- Add Layer 4: eKYC (Step-up Verification)
ALTER TABLE travel_rule_data
ADD COLUMN IF NOT EXISTS ekyc_required BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS ekyc_status VARCHAR(50),
ADD COLUMN IF NOT EXISTS ekyc_document_type VARCHAR(50),
ADD COLUMN IF NOT EXISTS ekyc_document_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS ekyc_verified_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS ekyc_provider VARCHAR(50),
ADD COLUMN IF NOT EXISTS ekyc_reference VARCHAR(255);

-- Add metadata for verification flow
ALTER TABLE travel_rule_data
ADD COLUMN IF NOT EXISTS verification_tier INT DEFAULT 1,
ADD COLUMN IF NOT EXISTS verification_completed_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS verification_notes TEXT;

-- Create indexes for verification queries
CREATE INDEX IF NOT EXISTS idx_travel_rule_signature_verified ON travel_rule_data(signature_verified);
CREATE INDEX IF NOT EXISTS idx_travel_rule_sanctions_status ON travel_rule_data(sanctions_check_status);
CREATE INDEX IF NOT EXISTS idx_travel_rule_wallet_sanctioned ON travel_rule_data(wallet_sanctioned);
CREATE INDEX IF NOT EXISTS idx_travel_rule_ekyc_status ON travel_rule_data(ekyc_status);
CREATE INDEX IF NOT EXISTS idx_travel_rule_geo_mismatch ON travel_rule_data(geo_mismatch_detected);
CREATE INDEX IF NOT EXISTS idx_travel_rule_verification_tier ON travel_rule_data(verification_tier);

-- Add constraints
ALTER TABLE travel_rule_data
ADD CONSTRAINT IF NOT EXISTS check_sanctions_status
    CHECK (sanctions_check_status IN ('pending', 'in_progress', 'completed', 'hit', 'cleared', 'failed'));

ALTER TABLE travel_rule_data
ADD CONSTRAINT IF NOT EXISTS check_sanctions_result
    CHECK (sanctions_check_result IS NULL OR sanctions_check_result IN ('clear', 'pep', 'sanctions', 'watchlist', 'adverse_media'));

ALTER TABLE travel_rule_data
ADD CONSTRAINT IF NOT EXISTS check_ekyc_status
    CHECK (ekyc_status IS NULL OR ekyc_status IN ('pending', 'in_progress', 'verified', 'failed', 'expired'));

ALTER TABLE travel_rule_data
ADD CONSTRAINT IF NOT EXISTS check_ekyc_document_type
    CHECK (ekyc_document_type IS NULL OR ekyc_document_type IN ('passport', 'national_id', 'drivers_license', 'residence_permit'));

ALTER TABLE travel_rule_data
ADD CONSTRAINT IF NOT EXISTS check_verification_tier
    CHECK (verification_tier BETWEEN 1 AND 4);

-- Add comments
COMMENT ON COLUMN travel_rule_data.payer_signature IS 'Layer 1: Cryptographic signature proving wallet ownership';
COMMENT ON COLUMN travel_rule_data.signature_message IS 'Layer 1: Message that was signed (includes declaration)';
COMMENT ON COLUMN travel_rule_data.signature_verified IS 'Layer 1: Whether signature verification passed';
COMMENT ON COLUMN travel_rule_data.signature_verified_at IS 'Layer 1: Timestamp of signature verification';

COMMENT ON COLUMN travel_rule_data.sanctions_check_status IS 'Layer 2: Status of sanctions screening';
COMMENT ON COLUMN travel_rule_data.sanctions_check_result IS 'Layer 2: Result of sanctions screening (clear, pep, sanctions, watchlist, adverse_media)';
COMMENT ON COLUMN travel_rule_data.sanctions_hit_details IS 'Layer 2: JSON details of sanctions hits (OFAC, UN lists)';
COMMENT ON COLUMN travel_rule_data.chainalysis_risk_score IS 'Layer 2: Risk score from Chainalysis/TRM Labs (0-100)';
COMMENT ON COLUMN travel_rule_data.wallet_sanctioned IS 'Layer 2: Whether wallet address is on sanctions list';

COMMENT ON COLUMN travel_rule_data.payer_ip_address IS 'Layer 3: IP address of payer during submission';
COMMENT ON COLUMN travel_rule_data.payer_ip_country IS 'Layer 3: Country derived from IP address';
COMMENT ON COLUMN travel_rule_data.payer_user_agent IS 'Layer 3: Browser user agent string';
COMMENT ON COLUMN travel_rule_data.payer_device_fingerprint IS 'Layer 3: Device fingerprint hash';
COMMENT ON COLUMN travel_rule_data.geo_mismatch_detected IS 'Layer 3: Whether payer country and IP country mismatch';
COMMENT ON COLUMN travel_rule_data.anomaly_score IS 'Layer 3: Anomaly detection score (0-100)';
COMMENT ON COLUMN travel_rule_data.anomaly_flags IS 'Layer 3: JSON array of detected anomalies';

COMMENT ON COLUMN travel_rule_data.ekyc_required IS 'Layer 4: Whether eKYC step-up verification is required';
COMMENT ON COLUMN travel_rule_data.ekyc_status IS 'Layer 4: Status of eKYC verification';
COMMENT ON COLUMN travel_rule_data.ekyc_document_type IS 'Layer 4: Type of identity document uploaded';
COMMENT ON COLUMN travel_rule_data.ekyc_document_id IS 'Layer 4: Document ID number (encrypted)';
COMMENT ON COLUMN travel_rule_data.ekyc_verified_at IS 'Layer 4: Timestamp of eKYC verification';
COMMENT ON COLUMN travel_rule_data.ekyc_provider IS 'Layer 4: eKYC service provider (FPT.AI, AWS Rekognition, etc)';
COMMENT ON COLUMN travel_rule_data.ekyc_reference IS 'Layer 4: Reference ID from eKYC provider';

COMMENT ON COLUMN travel_rule_data.verification_tier IS 'Highest verification tier reached (1-4): 1=Signature, 2=Sanctions, 3=Passive, 4=eKYC';
COMMENT ON COLUMN travel_rule_data.verification_completed_at IS 'Timestamp when all required verification completed';
COMMENT ON COLUMN travel_rule_data.verification_notes IS 'Internal notes about verification process';

-- Create view for verification status
CREATE OR REPLACE VIEW v_travel_rule_verification_status AS
SELECT
    id,
    payment_id,
    payer_full_name,
    payer_wallet_address,
    payer_country,
    transaction_amount,

    -- Layer 1: Signature
    signature_verified AS layer1_signature_verified,
    signature_verified_at AS layer1_verified_at,

    -- Layer 2: Sanctions
    sanctions_check_status AS layer2_status,
    sanctions_check_result AS layer2_result,
    wallet_sanctioned AS layer2_wallet_sanctioned,
    chainalysis_risk_score AS layer2_risk_score,

    -- Layer 3: Passive Signals
    geo_mismatch_detected AS layer3_geo_mismatch,
    anomaly_score AS layer3_anomaly_score,
    CASE
        WHEN anomaly_flags IS NOT NULL THEN jsonb_array_length(anomaly_flags)
        ELSE 0
    END AS layer3_anomaly_count,

    -- Layer 4: eKYC
    ekyc_required AS layer4_required,
    ekyc_status AS layer4_status,
    ekyc_verified_at AS layer4_verified_at,

    -- Overall status
    verification_tier,
    verification_completed_at,

    -- Risk flags
    CASE
        WHEN wallet_sanctioned THEN TRUE
        WHEN sanctions_check_result IN ('sanctions', 'pep', 'watchlist') THEN TRUE
        WHEN geo_mismatch_detected AND transaction_amount > 5000 THEN TRUE
        WHEN anomaly_score > 70 THEN TRUE
        ELSE FALSE
    END AS requires_manual_review,

    created_at,
    updated_at

FROM travel_rule_data
ORDER BY created_at DESC;

COMMENT ON VIEW v_travel_rule_verification_status IS 'Comprehensive view of 4-layer verification status for Travel Rule records';

-- Create function to calculate verification tier
CREATE OR REPLACE FUNCTION calculate_verification_tier(
    p_travel_rule_id UUID
)
RETURNS INT AS $$
DECLARE
    v_tier INT := 1;
    v_signature_verified BOOLEAN;
    v_sanctions_completed BOOLEAN;
    v_anomaly_checked BOOLEAN;
    v_ekyc_verified BOOLEAN;
BEGIN
    SELECT
        signature_verified,
        sanctions_check_status = 'completed',
        anomaly_score IS NOT NULL,
        ekyc_status = 'verified'
    INTO
        v_signature_verified,
        v_sanctions_completed,
        v_anomaly_checked,
        v_ekyc_verified
    FROM travel_rule_data
    WHERE id = p_travel_rule_id;

    -- Tier 1: Signature (required)
    IF v_signature_verified THEN
        v_tier := 1;
    END IF;

    -- Tier 2: Sanctions
    IF v_signature_verified AND v_sanctions_completed THEN
        v_tier := 2;
    END IF;

    -- Tier 3: Passive signals
    IF v_signature_verified AND v_sanctions_completed AND v_anomaly_checked THEN
        v_tier := 3;
    END IF;

    -- Tier 4: eKYC
    IF v_signature_verified AND v_sanctions_completed AND v_anomaly_checked AND v_ekyc_verified THEN
        v_tier := 4;
    END IF;

    RETURN v_tier;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION calculate_verification_tier IS 'Calculates the highest verification tier achieved for a Travel Rule record';

-- Create trigger to auto-update verification tier
CREATE OR REPLACE FUNCTION update_verification_tier()
RETURNS TRIGGER AS $$
BEGIN
    NEW.verification_tier := calculate_verification_tier(NEW.id);

    -- Mark verification as completed if all required checks pass
    IF NEW.signature_verified
       AND NEW.sanctions_check_status = 'completed'
       AND NEW.anomaly_score IS NOT NULL
       AND (NOT NEW.ekyc_required OR NEW.ekyc_status = 'verified') THEN
        NEW.verification_completed_at := NOW();
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_verification_tier ON travel_rule_data;
CREATE TRIGGER trigger_update_verification_tier
    BEFORE UPDATE ON travel_rule_data
    FOR EACH ROW
    WHEN (
        NEW.signature_verified IS DISTINCT FROM OLD.signature_verified OR
        NEW.sanctions_check_status IS DISTINCT FROM OLD.sanctions_check_status OR
        NEW.anomaly_score IS DISTINCT FROM OLD.anomaly_score OR
        NEW.ekyc_status IS DISTINCT FROM OLD.ekyc_status
    )
    EXECUTE FUNCTION update_verification_tier();
