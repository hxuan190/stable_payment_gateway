-- Migration: Create wallet_identity_mappings table (PRD v2.2 - Smart Identity Mapping)
-- Purpose: Map cryptocurrency wallet addresses to user identities for one-time KYC

CREATE TABLE IF NOT EXISTS wallet_identity_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Wallet Information
    wallet_address VARCHAR(255) NOT NULL,
    blockchain VARCHAR(20) NOT NULL, -- 'tron', 'solana', 'bsc', 'ethereum', 'polygon'
    wallet_address_hash VARCHAR(64) NOT NULL, -- SHA-256 hash of wallet_address for privacy

    -- User Link
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- KYC Status (denormalized for fast lookups)
    kyc_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    kyc_verified_at TIMESTAMP,

    -- Activity Tracking
    first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    payment_count INTEGER DEFAULT 0,
    total_volume_usd DECIMAL(20, 2) DEFAULT 0,
    last_payment_id UUID, -- Reference to last payment made with this wallet

    -- Risk Flags (for quick filtering)
    is_flagged BOOLEAN DEFAULT FALSE,
    flag_reason TEXT,
    flagged_at TIMESTAMP,

    -- Metadata
    metadata JSONB DEFAULT '{}', -- Additional data (e.g., wallet label, device info)

    -- Audit fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Unique constraint: one mapping per wallet address per blockchain
    CONSTRAINT unique_wallet_blockchain UNIQUE (wallet_address, blockchain)
);

-- Indexes for performance (CRITICAL for PRD v2.2 identity recognition)
CREATE UNIQUE INDEX idx_wallet_mappings_address_blockchain ON wallet_identity_mappings(wallet_address, blockchain);
CREATE INDEX idx_wallet_mappings_address_hash ON wallet_identity_mappings(wallet_address_hash);
CREATE INDEX idx_wallet_mappings_user_id ON wallet_identity_mappings(user_id);
CREATE INDEX idx_wallet_mappings_kyc_status ON wallet_identity_mappings(kyc_status);
CREATE INDEX idx_wallet_mappings_blockchain ON wallet_identity_mappings(blockchain);
CREATE INDEX idx_wallet_mappings_last_seen_at ON wallet_identity_mappings(last_seen_at DESC);
CREATE INDEX idx_wallet_mappings_is_flagged ON wallet_identity_mappings(is_flagged) WHERE is_flagged = TRUE;
CREATE INDEX idx_wallet_mappings_created_at ON wallet_identity_mappings(created_at DESC);

-- Partial index for active wallets (used in last 30 days)
CREATE INDEX idx_wallet_mappings_active ON wallet_identity_mappings(wallet_address, blockchain, user_id)
    WHERE last_seen_at > CURRENT_TIMESTAMP - INTERVAL '30 days';

-- Trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_wallet_mappings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_wallet_mappings_updated_at
    BEFORE UPDATE ON wallet_identity_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_wallet_mappings_updated_at();

-- Function to compute wallet address hash (SHA-256)
CREATE OR REPLACE FUNCTION compute_wallet_hash(address TEXT)
RETURNS VARCHAR(64) AS $$
BEGIN
    RETURN encode(digest(LOWER(address), 'sha256'), 'hex');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Trigger to automatically compute wallet_address_hash before insert/update
CREATE OR REPLACE FUNCTION set_wallet_address_hash()
RETURNS TRIGGER AS $$
BEGIN
    NEW.wallet_address_hash = compute_wallet_hash(NEW.wallet_address);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_wallet_address_hash
    BEFORE INSERT OR UPDATE OF wallet_address ON wallet_identity_mappings
    FOR EACH ROW
    EXECUTE FUNCTION set_wallet_address_hash();

-- Add check constraints
ALTER TABLE wallet_identity_mappings ADD CONSTRAINT check_wallet_kyc_status
    CHECK (kyc_status IN ('pending', 'in_progress', 'approved', 'rejected', 'expired'));

ALTER TABLE wallet_identity_mappings ADD CONSTRAINT check_wallet_blockchain
    CHECK (blockchain IN ('tron', 'solana', 'bsc', 'ethereum', 'polygon'));

ALTER TABLE wallet_identity_mappings ADD CONSTRAINT check_wallet_payment_count
    CHECK (payment_count >= 0);

ALTER TABLE wallet_identity_mappings ADD CONSTRAINT check_wallet_total_volume
    CHECK (total_volume_usd >= 0);

-- View: Active wallet mappings (used in last 7 days)
CREATE OR REPLACE VIEW active_wallet_mappings AS
SELECT
    wm.*,
    u.full_name AS user_name,
    u.email AS user_email,
    u.risk_level AS user_risk_level
FROM wallet_identity_mappings wm
JOIN users u ON wm.user_id = u.id
WHERE wm.last_seen_at > CURRENT_TIMESTAMP - INTERVAL '7 days'
    AND wm.kyc_status = 'approved'
    AND wm.is_flagged = FALSE;

-- View: Wallet activity summary
CREATE OR REPLACE VIEW wallet_activity_summary AS
SELECT
    wm.blockchain,
    COUNT(DISTINCT wm.wallet_address) AS total_wallets,
    COUNT(DISTINCT wm.user_id) AS total_users,
    SUM(wm.payment_count) AS total_payments,
    SUM(wm.total_volume_usd) AS total_volume_usd,
    COUNT(*) FILTER (WHERE wm.last_seen_at > CURRENT_TIMESTAMP - INTERVAL '24 hours') AS active_24h,
    COUNT(*) FILTER (WHERE wm.last_seen_at > CURRENT_TIMESTAMP - INTERVAL '7 days') AS active_7d,
    COUNT(*) FILTER (WHERE wm.is_flagged = TRUE) AS flagged_wallets
FROM wallet_identity_mappings wm
GROUP BY wm.blockchain;

-- Comments for documentation
COMMENT ON TABLE wallet_identity_mappings IS 'Maps cryptocurrency wallet addresses to user identities for one-time KYC (PRD v2.2 Smart Identity Mapping)';
COMMENT ON COLUMN wallet_identity_mappings.wallet_address IS 'Original wallet address (plain text for lookups)';
COMMENT ON COLUMN wallet_identity_mappings.wallet_address_hash IS 'SHA-256 hash of wallet address for privacy/security';
COMMENT ON COLUMN wallet_identity_mappings.payment_count IS 'Total number of payments made with this wallet';
COMMENT ON COLUMN wallet_identity_mappings.total_volume_usd IS 'Cumulative payment volume in USD for AML monitoring';
COMMENT ON COLUMN wallet_identity_mappings.is_flagged IS 'Manual flag by compliance team for suspicious activity';
COMMENT ON VIEW active_wallet_mappings IS 'Wallets that made payments in last 7 days (for Redis caching)';
COMMENT ON VIEW wallet_activity_summary IS 'Aggregated wallet activity statistics by blockchain';
