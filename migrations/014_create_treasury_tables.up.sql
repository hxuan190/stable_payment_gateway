-- Migration: Create treasury tables (PRD v2.2 - Custodial Treasury)
-- Purpose: Manage hot/cold wallets and automated sweeping operations

-- Table: treasury_wallets - Hot and cold wallet inventory
CREATE TABLE IF NOT EXISTS treasury_wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Wallet Type
    wallet_type VARCHAR(20) NOT NULL, -- 'hot', 'cold'

    -- Blockchain Information
    blockchain VARCHAR(20) NOT NULL, -- 'tron', 'solana', 'bsc', 'ethereum', 'polygon'
    address VARCHAR(255) NOT NULL,
    address_hash VARCHAR(64) NOT NULL, -- SHA-256 hash for privacy

    -- Multi-Sig Configuration (for cold wallets)
    is_multisig BOOLEAN DEFAULT FALSE,
    multisig_scheme VARCHAR(20), -- '2-of-3', '3-of-5', etc.
    multisig_signers TEXT[], -- Array of signer addresses
    multisig_threshold INTEGER, -- Number of signatures required

    -- Balance Tracking
    balance_crypto DECIMAL(30, 18) DEFAULT 0, -- Current balance in native token/coin
    balance_usd DECIMAL(20, 2) DEFAULT 0, -- USD equivalent
    balance_last_updated_at TIMESTAMP,

    -- Sweeping Configuration (for hot wallets)
    sweep_threshold_usd DECIMAL(20, 2) DEFAULT 10000, -- Sweep when balance > this
    sweep_target_wallet_id UUID, -- Target cold wallet for sweeping
    sweep_buffer_usd DECIMAL(20, 2) DEFAULT 5000, -- Leave this much in hot wallet after sweep
    auto_sweep_enabled BOOLEAN DEFAULT TRUE,

    -- Activity Tracking
    last_swept_at TIMESTAMP,
    last_deposit_at TIMESTAMP,
    last_withdrawal_at TIMESTAMP,
    total_swept_amount_usd DECIMAL(20, 2) DEFAULT 0,

    -- Status
    status VARCHAR(20) DEFAULT 'active', -- 'active', 'suspended', 'deprecated'
    is_monitored BOOLEAN DEFAULT TRUE, -- Whether to actively monitor this wallet

    -- Security
    encrypted_private_key TEXT, -- Encrypted private key (for hot wallets only, NULL for cold)
    encryption_method VARCHAR(50), -- 'AWS-KMS', 'Vault', 'GPG'
    key_storage_location VARCHAR(100), -- Reference to where key is stored

    -- Metadata
    label VARCHAR(255), -- Human-readable label (e.g., "TRON Hot Wallet #1")
    description TEXT,
    metadata JSONB DEFAULT '{}',

    -- Audit fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(100),
    updated_by VARCHAR(100),

    -- Unique constraint
    CONSTRAINT unique_treasury_wallet_address UNIQUE (blockchain, address)
);

-- Indexes for treasury_wallets
CREATE UNIQUE INDEX idx_treasury_wallets_blockchain_address ON treasury_wallets(blockchain, address);
CREATE INDEX idx_treasury_wallets_type ON treasury_wallets(wallet_type);
CREATE INDEX idx_treasury_wallets_blockchain ON treasury_wallets(blockchain);
CREATE INDEX idx_treasury_wallets_status ON treasury_wallets(status) WHERE status = 'active';
CREATE INDEX idx_treasury_wallets_auto_sweep ON treasury_wallets(auto_sweep_enabled, wallet_type)
    WHERE auto_sweep_enabled = TRUE AND wallet_type = 'hot';
CREATE INDEX idx_treasury_wallets_balance_updated ON treasury_wallets(balance_last_updated_at DESC);
CREATE INDEX idx_treasury_wallets_label ON treasury_wallets(label);

-- Table: treasury_operations - Log of all treasury operations
CREATE TABLE IF NOT EXISTS treasury_operations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Operation Type
    operation_type VARCHAR(50) NOT NULL, -- 'sweep', 'manual_transfer', 'otc_settlement', 'consolidation', 'emergency_withdrawal'

    -- Wallet Information
    from_wallet_id UUID REFERENCES treasury_wallets(id) ON DELETE SET NULL,
    to_wallet_id UUID REFERENCES treasury_wallets(id) ON DELETE SET NULL,
    from_address VARCHAR(255),
    to_address VARCHAR(255),

    -- Blockchain & Transaction
    blockchain VARCHAR(20) NOT NULL,
    amount DECIMAL(30, 18) NOT NULL, -- Amount transferred
    currency VARCHAR(10) NOT NULL, -- 'TRX', 'SOL', 'BNB', 'USDT', 'USDC'
    amount_usd DECIMAL(20, 2), -- USD equivalent at time of operation
    tx_hash VARCHAR(255), -- Blockchain transaction hash
    tx_confirmations INTEGER DEFAULT 0,

    -- Multi-Sig (if applicable)
    requires_multisig BOOLEAN DEFAULT FALSE,
    signatures_required INTEGER,
    signatures_collected INTEGER DEFAULT 0,
    signers JSONB, -- Array of {signer_address, signed_at, signature}

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'initiated', -- 'initiated', 'pending_signatures', 'broadcasted', 'confirmed', 'failed', 'cancelled'
    initiated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    broadcasted_at TIMESTAMP,
    confirmed_at TIMESTAMP,
    failed_at TIMESTAMP,
    cancelled_at TIMESTAMP,

    -- Reason & Context
    reason TEXT, -- Why this operation was triggered
    triggered_by VARCHAR(50), -- 'auto_sweep', 'manual', 'threshold', 'scheduled'
    initiated_by VARCHAR(100), -- User/system who initiated

    -- Error Handling
    error_message TEXT,
    error_code VARCHAR(50),
    retry_count INTEGER DEFAULT 0,

    -- Gas/Fee
    gas_used DECIMAL(30, 18),
    gas_price DECIMAL(30, 18),
    transaction_fee DECIMAL(30, 18),
    transaction_fee_usd DECIMAL(20, 2),

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Audit fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(100),
    updated_by VARCHAR(100)
);

-- Indexes for treasury_operations
CREATE INDEX idx_treasury_operations_from_wallet ON treasury_operations(from_wallet_id);
CREATE INDEX idx_treasury_operations_to_wallet ON treasury_operations(to_wallet_id);
CREATE INDEX idx_treasury_operations_type ON treasury_operations(operation_type);
CREATE INDEX idx_treasury_operations_blockchain ON treasury_operations(blockchain);
CREATE INDEX idx_treasury_operations_tx_hash ON treasury_operations(tx_hash) WHERE tx_hash IS NOT NULL;
CREATE INDEX idx_treasury_operations_status ON treasury_operations(status);
CREATE INDEX idx_treasury_operations_created_at ON treasury_operations(created_at DESC);
CREATE INDEX idx_treasury_operations_confirmed_at ON treasury_operations(confirmed_at DESC) WHERE confirmed_at IS NOT NULL;

-- Composite index for sweep monitoring
CREATE INDEX idx_treasury_operations_sweep_monitoring ON treasury_operations(operation_type, status, initiated_at DESC)
    WHERE operation_type = 'sweep';

-- Partial index for pending multisig operations
CREATE INDEX idx_treasury_operations_pending_multisig ON treasury_operations(requires_multisig, signatures_required, signatures_collected, status)
    WHERE requires_multisig = TRUE AND status = 'pending_signatures';

-- Trigger to update updated_at for treasury_wallets
CREATE OR REPLACE FUNCTION update_treasury_wallets_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_treasury_wallets_updated_at
    BEFORE UPDATE ON treasury_wallets
    FOR EACH ROW
    EXECUTE FUNCTION update_treasury_wallets_updated_at();

-- Trigger to update updated_at for treasury_operations
CREATE OR REPLACE FUNCTION update_treasury_operations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_treasury_operations_updated_at
    BEFORE UPDATE ON treasury_operations
    FOR EACH ROW
    EXECUTE FUNCTION update_treasury_operations_updated_at();

-- Trigger to compute address_hash automatically
CREATE OR REPLACE FUNCTION set_treasury_wallet_address_hash()
RETURNS TRIGGER AS $$
BEGIN
    NEW.address_hash = encode(digest(LOWER(NEW.address), 'sha256'), 'hex');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_treasury_wallet_address_hash
    BEFORE INSERT OR UPDATE OF address ON treasury_wallets
    FOR EACH ROW
    EXECUTE FUNCTION set_treasury_wallet_address_hash();

-- Check constraints
ALTER TABLE treasury_wallets ADD CONSTRAINT check_treasury_wallet_type
    CHECK (wallet_type IN ('hot', 'cold'));

ALTER TABLE treasury_wallets ADD CONSTRAINT check_treasury_wallet_blockchain
    CHECK (blockchain IN ('tron', 'solana', 'bsc', 'ethereum', 'polygon'));

ALTER TABLE treasury_wallets ADD CONSTRAINT check_treasury_wallet_status
    CHECK (status IN ('active', 'suspended', 'deprecated'));

ALTER TABLE treasury_wallets ADD CONSTRAINT check_treasury_balance_positive
    CHECK (balance_crypto >= 0 AND balance_usd >= 0);

ALTER TABLE treasury_wallets ADD CONSTRAINT check_treasury_sweep_amounts
    CHECK (sweep_threshold_usd IS NULL OR sweep_threshold_usd > sweep_buffer_usd);

ALTER TABLE treasury_operations ADD CONSTRAINT check_treasury_op_type
    CHECK (operation_type IN ('sweep', 'manual_transfer', 'otc_settlement', 'consolidation', 'emergency_withdrawal'));

ALTER TABLE treasury_operations ADD CONSTRAINT check_treasury_op_status
    CHECK (status IN ('initiated', 'pending_signatures', 'broadcasted', 'confirmed', 'failed', 'cancelled'));

ALTER TABLE treasury_operations ADD CONSTRAINT check_treasury_op_amount_positive
    CHECK (amount > 0);

-- View: Hot wallets needing sweep
CREATE OR REPLACE VIEW hot_wallets_needing_sweep AS
SELECT
    tw.id,
    tw.blockchain,
    tw.address,
    tw.label,
    tw.balance_usd,
    tw.sweep_threshold_usd,
    tw.sweep_buffer_usd,
    tw.balance_usd - tw.sweep_buffer_usd AS sweep_amount_usd,
    tw.last_swept_at,
    target.address AS target_cold_wallet_address
FROM treasury_wallets tw
LEFT JOIN treasury_wallets target ON tw.sweep_target_wallet_id = target.id
WHERE tw.wallet_type = 'hot'
    AND tw.auto_sweep_enabled = TRUE
    AND tw.status = 'active'
    AND tw.balance_usd >= tw.sweep_threshold_usd;

-- View: Treasury operation summary
CREATE OR REPLACE VIEW treasury_operation_summary AS
SELECT
    operation_type,
    blockchain,
    COUNT(*) AS total_operations,
    COUNT(*) FILTER (WHERE status = 'confirmed') AS confirmed_count,
    COUNT(*) FILTER (WHERE status = 'failed') AS failed_count,
    COUNT(*) FILTER (WHERE status = 'pending_signatures') AS pending_multisig_count,
    SUM(amount_usd) FILTER (WHERE status = 'confirmed') AS total_amount_usd,
    AVG(transaction_fee_usd) FILTER (WHERE status = 'confirmed') AS avg_fee_usd,
    MAX(confirmed_at) AS last_operation_at
FROM treasury_operations
GROUP BY operation_type, blockchain;

-- View: Treasury balance summary
CREATE OR REPLACE VIEW treasury_balance_summary AS
SELECT
    blockchain,
    wallet_type,
    COUNT(*) AS wallet_count,
    SUM(balance_crypto) AS total_balance_crypto,
    SUM(balance_usd) AS total_balance_usd,
    AVG(balance_usd) AS avg_balance_usd,
    MAX(balance_last_updated_at) AS last_updated
FROM treasury_wallets
WHERE status = 'active'
GROUP BY blockchain, wallet_type;

-- Comments for documentation
COMMENT ON TABLE treasury_wallets IS 'Inventory of hot and cold cryptocurrency wallets (PRD v2.2 Custodial Treasury)';
COMMENT ON TABLE treasury_operations IS 'Log of all treasury operations including auto-sweeping and manual transfers';
COMMENT ON COLUMN treasury_wallets.wallet_type IS 'Wallet type: hot (online, active use) or cold (offline, long-term storage)';
COMMENT ON COLUMN treasury_wallets.sweep_threshold_usd IS 'Automatically sweep hot wallet when balance exceeds this USD amount';
COMMENT ON COLUMN treasury_wallets.sweep_buffer_usd IS 'Amount to leave in hot wallet after sweeping';
COMMENT ON COLUMN treasury_wallets.multisig_scheme IS 'Multi-signature scheme like 2-of-3 (2 signatures required out of 3 signers)';
COMMENT ON COLUMN treasury_operations.operation_type IS 'Type of treasury operation: sweep (auto), manual_transfer, otc_settlement, etc.';
COMMENT ON COLUMN treasury_operations.status IS 'Operation status: initiated → pending_signatures → broadcasted → confirmed | failed';
COMMENT ON VIEW hot_wallets_needing_sweep IS 'Hot wallets with balance >= threshold that need to be swept to cold storage';
COMMENT ON VIEW treasury_operation_summary IS 'Summary statistics of treasury operations by type and blockchain';
COMMENT ON VIEW treasury_balance_summary IS 'Aggregated balance across all wallets by blockchain and type';
