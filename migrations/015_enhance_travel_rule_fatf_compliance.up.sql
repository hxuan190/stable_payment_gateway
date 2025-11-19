-- Migration 015: Enhance Travel Rule for Full FATF Compliance
-- Adds additional fields required by FATF Travel Rule recommendations
-- Reference: FATF Recommendation 16 (Wire Transfers)

-- Add beneficiary (merchant) wallet address
ALTER TABLE travel_rule_data
ADD COLUMN IF NOT EXISTS merchant_wallet_address VARCHAR(255),
ADD COLUMN IF NOT EXISTS merchant_id_document VARCHAR(255),
ADD COLUMN IF NOT EXISTS merchant_business_registration VARCHAR(255);

-- Add transaction purpose and additional identifiers
ALTER TABLE travel_rule_data
ADD COLUMN IF NOT EXISTS transaction_purpose VARCHAR(100),
ADD COLUMN IF NOT EXISTS payer_date_of_birth DATE,
ADD COLUMN IF NOT EXISTS payer_address TEXT,
ADD COLUMN IF NOT EXISTS merchant_address TEXT;

-- Add risk assessment fields
ALTER TABLE travel_rule_data
ADD COLUMN IF NOT EXISTS risk_level VARCHAR(20) DEFAULT 'low',
ADD COLUMN IF NOT EXISTS risk_score DECIMAL(5, 2),
ADD COLUMN IF NOT EXISTS screening_status VARCHAR(50) DEFAULT 'pending',
ADD COLUMN IF NOT EXISTS screening_completed_at TIMESTAMP;

-- Add regulatory reporting fields
ALTER TABLE travel_rule_data
ADD COLUMN IF NOT EXISTS reported_to_authority BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS reported_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS report_reference VARCHAR(255);

-- Add metadata for data retention and archival
ALTER TABLE travel_rule_data
ADD COLUMN IF NOT EXISTS retention_policy VARCHAR(50) DEFAULT 'standard',
ADD COLUMN IF NOT EXISTS archived_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

-- Create indexes for new fields
CREATE INDEX IF NOT EXISTS idx_travel_rule_merchant_wallet ON travel_rule_data(merchant_wallet_address);
CREATE INDEX IF NOT EXISTS idx_travel_rule_risk_level ON travel_rule_data(risk_level);
CREATE INDEX IF NOT EXISTS idx_travel_rule_screening_status ON travel_rule_data(screening_status);
CREATE INDEX IF NOT EXISTS idx_travel_rule_reported ON travel_rule_data(reported_to_authority, reported_at);
CREATE INDEX IF NOT EXISTS idx_travel_rule_transaction_purpose ON travel_rule_data(transaction_purpose);

-- Add constraint for risk level
ALTER TABLE travel_rule_data
ADD CONSTRAINT IF NOT EXISTS check_travel_rule_risk_level
    CHECK (risk_level IN ('low', 'medium', 'high', 'critical'));

-- Add constraint for screening status
ALTER TABLE travel_rule_data
ADD CONSTRAINT IF NOT EXISTS check_travel_rule_screening_status
    CHECK (screening_status IN ('pending', 'in_progress', 'completed', 'flagged', 'cleared'));

-- Add constraint for transaction purpose
ALTER TABLE travel_rule_data
ADD CONSTRAINT IF NOT EXISTS check_travel_rule_transaction_purpose
    CHECK (transaction_purpose IN (
        'payment_for_goods',
        'payment_for_services',
        'hotel_accommodation',
        'restaurant_payment',
        'tourism_service',
        'transport_service',
        'entertainment',
        'other'
    ));

-- Add constraint for retention policy
ALTER TABLE travel_rule_data
ADD CONSTRAINT IF NOT EXISTS check_travel_rule_retention_policy
    CHECK (retention_policy IN ('standard', 'extended', 'permanent'));

-- Add comments for new fields
COMMENT ON COLUMN travel_rule_data.merchant_wallet_address IS 'Blockchain wallet address of the beneficiary (merchant)';
COMMENT ON COLUMN travel_rule_data.merchant_id_document IS 'Merchant ID document number for enhanced due diligence';
COMMENT ON COLUMN travel_rule_data.merchant_business_registration IS 'Merchant business registration number';
COMMENT ON COLUMN travel_rule_data.transaction_purpose IS 'Purpose of the transaction for regulatory reporting';
COMMENT ON COLUMN travel_rule_data.payer_date_of_birth IS 'Date of birth of the payer for identity verification';
COMMENT ON COLUMN travel_rule_data.payer_address IS 'Full address of the payer';
COMMENT ON COLUMN travel_rule_data.merchant_address IS 'Full address of the merchant';
COMMENT ON COLUMN travel_rule_data.risk_level IS 'AML risk level: low, medium, high, critical';
COMMENT ON COLUMN travel_rule_data.risk_score IS 'Numerical risk score (0-100)';
COMMENT ON COLUMN travel_rule_data.screening_status IS 'Status of AML/compliance screening';
COMMENT ON COLUMN travel_rule_data.screening_completed_at IS 'Timestamp when screening was completed';
COMMENT ON COLUMN travel_rule_data.reported_to_authority IS 'Whether this transaction was reported to regulatory authority';
COMMENT ON COLUMN travel_rule_data.reported_at IS 'Timestamp when reported to authority';
COMMENT ON COLUMN travel_rule_data.report_reference IS 'Reference number from regulatory authority report';
COMMENT ON COLUMN travel_rule_data.retention_policy IS 'Data retention policy: standard (7 years), extended (10 years), permanent';
COMMENT ON COLUMN travel_rule_data.archived_at IS 'Timestamp when data was archived to cold storage';
COMMENT ON COLUMN travel_rule_data.updated_at IS 'Timestamp of last update';

-- Create a view for regulatory reporting
CREATE OR REPLACE VIEW v_travel_rule_regulatory_report AS
SELECT
    t.id,
    t.payment_id,
    t.created_at AS transaction_date,

    -- Originator (Payer) Information
    t.payer_full_name AS originator_name,
    t.payer_wallet_address AS originator_wallet,
    t.payer_country AS originator_country,
    t.payer_id_document AS originator_id,
    t.payer_date_of_birth AS originator_dob,
    t.payer_address AS originator_address,

    -- Beneficiary (Merchant) Information
    t.merchant_full_name AS beneficiary_name,
    t.merchant_wallet_address AS beneficiary_wallet,
    t.merchant_country AS beneficiary_country,
    t.merchant_id_document AS beneficiary_id,
    t.merchant_business_registration AS beneficiary_business_reg,
    t.merchant_address AS beneficiary_address,

    -- Transaction Details
    t.transaction_amount,
    t.transaction_currency,
    t.transaction_purpose,

    -- Risk Assessment
    t.risk_level,
    t.risk_score,
    t.screening_status,
    t.screening_completed_at,

    -- Reporting Status
    t.reported_to_authority,
    t.reported_at,
    t.report_reference,

    -- Cross-border Flag
    CASE
        WHEN t.payer_country != t.merchant_country THEN TRUE
        ELSE FALSE
    END AS is_cross_border

FROM travel_rule_data t
WHERE t.transaction_amount >= 1000 -- FATF threshold
ORDER BY t.created_at DESC;

COMMENT ON VIEW v_travel_rule_regulatory_report IS 'View for generating FATF Travel Rule compliance reports for regulatory authorities';

-- Create a function to auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_travel_rule_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for updated_at
DROP TRIGGER IF EXISTS trigger_travel_rule_updated_at ON travel_rule_data;
CREATE TRIGGER trigger_travel_rule_updated_at
    BEFORE UPDATE ON travel_rule_data
    FOR EACH ROW
    EXECUTE FUNCTION update_travel_rule_updated_at();

-- Create a function to check Travel Rule compliance
CREATE OR REPLACE FUNCTION check_travel_rule_compliance(
    p_transaction_amount DECIMAL,
    p_payer_full_name VARCHAR,
    p_merchant_full_name VARCHAR,
    p_payer_country CHAR,
    p_merchant_country CHAR
)
RETURNS TABLE(
    is_compliant BOOLEAN,
    requires_travel_rule BOOLEAN,
    missing_fields TEXT[]
) AS $$
DECLARE
    v_threshold DECIMAL := 1000;
    v_missing_fields TEXT[] := ARRAY[]::TEXT[];
BEGIN
    -- Check if Travel Rule applies (amount >= $1000)
    requires_travel_rule := p_transaction_amount >= v_threshold;

    IF NOT requires_travel_rule THEN
        is_compliant := TRUE;
        RETURN NEXT;
        RETURN;
    END IF;

    -- Check required fields
    IF p_payer_full_name IS NULL OR LENGTH(TRIM(p_payer_full_name)) < 2 THEN
        v_missing_fields := array_append(v_missing_fields, 'payer_full_name');
    END IF;

    IF p_merchant_full_name IS NULL OR LENGTH(TRIM(p_merchant_full_name)) < 2 THEN
        v_missing_fields := array_append(v_missing_fields, 'merchant_full_name');
    END IF;

    IF p_payer_country IS NULL OR LENGTH(p_payer_country) != 2 THEN
        v_missing_fields := array_append(v_missing_fields, 'payer_country');
    END IF;

    IF p_merchant_country IS NULL OR LENGTH(p_merchant_country) != 2 THEN
        v_missing_fields := array_append(v_missing_fields, 'merchant_country');
    END IF;

    -- Determine compliance
    is_compliant := (array_length(v_missing_fields, 1) IS NULL);
    missing_fields := v_missing_fields;

    RETURN NEXT;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION check_travel_rule_compliance IS 'Validates whether a transaction meets FATF Travel Rule compliance requirements';
