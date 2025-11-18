-- Remove KYC Tier columns from merchants table
DROP INDEX IF EXISTS idx_merchants_volume_reset;
DROP INDEX IF EXISTS idx_merchants_kyc_tier;

ALTER TABLE merchants
DROP COLUMN IF EXISTS volume_last_reset_at,
DROP COLUMN IF EXISTS total_volume_this_month_usd,
DROP COLUMN IF EXISTS monthly_limit_usd,
DROP COLUMN IF EXISTS kyc_tier;
