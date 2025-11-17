-- Create wallet_balance_snapshots table for tracking hot wallet balances over time
CREATE TABLE IF NOT EXISTS wallet_balance_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Blockchain details
    chain VARCHAR(20) NOT NULL CHECK (chain IN ('solana', 'bsc', 'ethereum')),
    network VARCHAR(20) NOT NULL CHECK (network IN ('mainnet', 'testnet', 'devnet')),

    -- Wallet information
    wallet_address VARCHAR(255) NOT NULL,

    -- Native token balance (SOL, BNB, ETH)
    native_balance DECIMAL(36, 18) NOT NULL DEFAULT 0,
    native_currency VARCHAR(10) NOT NULL,

    -- USDT balance
    usdt_balance DECIMAL(36, 18) NOT NULL DEFAULT 0,
    usdt_mint VARCHAR(255),

    -- USDC balance
    usdc_balance DECIMAL(36, 18) NOT NULL DEFAULT 0,
    usdc_mint VARCHAR(255),

    -- Total USD value (approximate)
    total_usd_value DECIMAL(36, 18) NOT NULL DEFAULT 0,

    -- Alert thresholds
    min_threshold_usd DECIMAL(36, 18) NOT NULL DEFAULT 1000,
    max_threshold_usd DECIMAL(36, 18) NOT NULL DEFAULT 10000,

    -- Alert status
    is_below_min_threshold BOOLEAN NOT NULL DEFAULT FALSE,
    is_above_max_threshold BOOLEAN NOT NULL DEFAULT FALSE,
    alert_sent BOOLEAN NOT NULL DEFAULT FALSE,
    alert_sent_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    snapshot_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT positive_native_balance CHECK (native_balance >= 0),
    CONSTRAINT positive_usdt_balance CHECK (usdt_balance >= 0),
    CONSTRAINT positive_usdc_balance CHECK (usdc_balance >= 0),
    CONSTRAINT positive_total_usd CHECK (total_usd_value >= 0),
    CONSTRAINT positive_thresholds CHECK (min_threshold_usd >= 0 AND max_threshold_usd > min_threshold_usd)
);

-- Create indexes for efficient querying
CREATE INDEX idx_wallet_balance_snapshots_chain_network ON wallet_balance_snapshots(chain, network);
CREATE INDEX idx_wallet_balance_snapshots_wallet_address ON wallet_balance_snapshots(wallet_address);
CREATE INDEX idx_wallet_balance_snapshots_snapshot_at ON wallet_balance_snapshots(snapshot_at DESC);
CREATE INDEX idx_wallet_balance_snapshots_alert_status ON wallet_balance_snapshots(is_below_min_threshold, is_above_max_threshold, alert_sent);
CREATE INDEX idx_wallet_balance_snapshots_created_at ON wallet_balance_snapshots(created_at DESC);

-- Create composite index for common queries
CREATE INDEX idx_wallet_balance_snapshots_chain_wallet_snapshot ON wallet_balance_snapshots(chain, wallet_address, snapshot_at DESC);

-- Add comments for documentation
COMMENT ON TABLE wallet_balance_snapshots IS 'Point-in-time snapshots of hot wallet balances for monitoring and alerting';
COMMENT ON COLUMN wallet_balance_snapshots.chain IS 'Blockchain network (solana, bsc, ethereum)';
COMMENT ON COLUMN wallet_balance_snapshots.network IS 'Network type (mainnet, testnet, devnet)';
COMMENT ON COLUMN wallet_balance_snapshots.wallet_address IS 'Hot wallet address being monitored';
COMMENT ON COLUMN wallet_balance_snapshots.native_balance IS 'Balance of native token (SOL, BNB, ETH)';
COMMENT ON COLUMN wallet_balance_snapshots.usdt_balance IS 'Balance of USDT stablecoin';
COMMENT ON COLUMN wallet_balance_snapshots.usdc_balance IS 'Balance of USDC stablecoin';
COMMENT ON COLUMN wallet_balance_snapshots.total_usd_value IS 'Approximate total value in USD';
COMMENT ON COLUMN wallet_balance_snapshots.min_threshold_usd IS 'Minimum threshold for low balance alert';
COMMENT ON COLUMN wallet_balance_snapshots.max_threshold_usd IS 'Maximum threshold for high balance alert';
COMMENT ON COLUMN wallet_balance_snapshots.is_below_min_threshold IS 'True if balance is below minimum threshold';
COMMENT ON COLUMN wallet_balance_snapshots.is_above_max_threshold IS 'True if balance is above maximum threshold';
COMMENT ON COLUMN wallet_balance_snapshots.alert_sent IS 'True if alert notification has been sent';
COMMENT ON COLUMN wallet_balance_snapshots.snapshot_at IS 'Time when the balance snapshot was taken';
