-- Create merchants table
CREATE TABLE IF NOT EXISTS merchants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    business_name VARCHAR(255) NOT NULL,
    business_tax_id VARCHAR(50),
    business_license_number VARCHAR(100),
    owner_full_name VARCHAR(255) NOT NULL,
    owner_id_card VARCHAR(50),
    phone_number VARCHAR(20),

    -- Address information
    business_address TEXT,
    city VARCHAR(100),
    district VARCHAR(100),

    -- Bank account information for payouts
    bank_account_name VARCHAR(255),
    bank_account_number VARCHAR(50),
    bank_name VARCHAR(100),
    bank_branch VARCHAR(100),

    -- KYC status: pending, approved, rejected, suspended
    kyc_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    kyc_submitted_at TIMESTAMP,
    kyc_approved_at TIMESTAMP,
    kyc_approved_by UUID,
    kyc_rejection_reason TEXT,

    -- API credentials
    api_key VARCHAR(255) UNIQUE,
    api_key_created_at TIMESTAMP,

    -- Webhook configuration
    webhook_url TEXT,
    webhook_secret VARCHAR(255),
    webhook_events TEXT[], -- Array of subscribed events

    -- Status: active, suspended, closed
    status VARCHAR(20) NOT NULL DEFAULT 'active',

    -- Metadata
    metadata JSONB,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Create index on email for faster lookups
CREATE INDEX idx_merchants_email ON merchants(email) WHERE deleted_at IS NULL;

-- Create index on api_key for authentication
CREATE INDEX idx_merchants_api_key ON merchants(api_key) WHERE deleted_at IS NULL AND api_key IS NOT NULL;

-- Create index on kyc_status for admin filtering
CREATE INDEX idx_merchants_kyc_status ON merchants(kyc_status) WHERE deleted_at IS NULL;

-- Create index on status for active merchants
CREATE INDEX idx_merchants_status ON merchants(status) WHERE deleted_at IS NULL;

-- Add constraint to ensure KYC status is valid
ALTER TABLE merchants ADD CONSTRAINT check_kyc_status
    CHECK (kyc_status IN ('pending', 'approved', 'rejected', 'suspended'));

-- Add constraint to ensure status is valid
ALTER TABLE merchants ADD CONSTRAINT check_merchant_status
    CHECK (status IN ('active', 'suspended', 'closed'));

-- Create function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to automatically update updated_at on merchants
CREATE TRIGGER update_merchants_updated_at
    BEFORE UPDATE ON merchants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE merchants IS 'Stores merchant information, KYC status, and API credentials';
COMMENT ON COLUMN merchants.kyc_status IS 'KYC verification status: pending, approved, rejected, suspended';
COMMENT ON COLUMN merchants.status IS 'Merchant account status: active, suspended, closed';
COMMENT ON COLUMN merchants.api_key IS 'API key for merchant authentication (generated after KYC approval)';
COMMENT ON COLUMN merchants.webhook_url IS 'URL where payment notifications will be sent';
COMMENT ON COLUMN merchants.metadata IS 'Additional merchant metadata as JSON';
