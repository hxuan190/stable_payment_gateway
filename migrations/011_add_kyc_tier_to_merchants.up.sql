-- Add KYC Tier and monthly limit tracking to merchants table
-- Supports compliance with Vietnamese cryptocurrency regulations

-- Add KYC Tier columns
ALTER TABLE merchants
ADD COLUMN kyc_tier VARCHAR(10) NOT NULL DEFAULT 'tier1'
    CHECK (kyc_tier IN ('tier1', 'tier2', 'tier3')),
ADD COLUMN monthly_limit_usd DECIMAL(20, 2) NOT NULL DEFAULT 5000.00,
ADD COLUMN total_volume_this_month_usd DECIMAL(20, 2) NOT NULL DEFAULT 0.00,
ADD COLUMN volume_last_reset_at TIMESTAMP NOT NULL DEFAULT NOW();

-- Create index for efficient filtering by tier
CREATE INDEX idx_merchants_kyc_tier ON merchants(kyc_tier) WHERE deleted_at IS NULL;

-- Create index for monthly volume tracking queries
CREATE INDEX idx_merchants_volume_reset ON merchants(volume_last_reset_at) WHERE deleted_at IS NULL;

-- Add comments for documentation
COMMENT ON COLUMN merchants.kyc_tier IS 'KYC tier level: tier1 ($5K/month), tier2 ($50K/month), tier3 ($500K/month)';
COMMENT ON COLUMN merchants.monthly_limit_usd IS 'Monthly transaction limit in USD based on KYC tier';
COMMENT ON COLUMN merchants.total_volume_this_month_usd IS 'Total transaction volume in current month (USD equivalent)';
COMMENT ON COLUMN merchants.volume_last_reset_at IS 'Last time monthly volume was reset (first day of month)';
