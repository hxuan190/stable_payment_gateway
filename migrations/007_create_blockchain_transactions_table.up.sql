-- Create blockchain_transactions table
-- This table tracks blockchain transaction details separately from payments
-- for better separation of concerns and blockchain-specific data
CREATE TABLE IF NOT EXISTS blockchain_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Blockchain details
    chain VARCHAR(20) NOT NULL, -- solana, bsc, ethereum
    network VARCHAR(20) NOT NULL, -- mainnet, testnet, devnet

    -- Transaction identification
    tx_hash VARCHAR(255) NOT NULL UNIQUE,
    block_number BIGINT,
    block_timestamp TIMESTAMP,

    -- Transaction details
    from_address VARCHAR(255) NOT NULL,
    to_address VARCHAR(255) NOT NULL,
    amount DECIMAL(20, 8) NOT NULL,
    currency VARCHAR(10) NOT NULL, -- USDT, USDC, etc.
    token_mint VARCHAR(255), -- For SPL tokens on Solana or contract address on BSC/ETH

    -- Transaction memo/reference
    memo TEXT,
    parsed_payment_reference VARCHAR(100), -- Extracted from memo

    -- Confirmation status
    confirmations INT NOT NULL DEFAULT 0,
    is_finalized BOOLEAN NOT NULL DEFAULT FALSE,
    finalized_at TIMESTAMP,

    -- Gas/fee information
    gas_used BIGINT,
    gas_price DECIMAL(20, 8),
    transaction_fee DECIMAL(20, 8),
    fee_currency VARCHAR(10), -- SOL, BNB, ETH

    -- Associated payment (if matched)
    payment_id UUID,
    is_matched BOOLEAN NOT NULL DEFAULT FALSE,
    matched_at TIMESTAMP,

    -- Transaction status
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, confirmed, finalized, failed

    -- Raw transaction data (for debugging)
    raw_transaction JSONB,

    -- Error tracking
    error_message TEXT,
    error_details JSONB,

    -- Metadata
    metadata JSONB,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Foreign key constraint (optional, payment might not exist yet when tx detected)
    CONSTRAINT fk_blockchain_tx_payment
        FOREIGN KEY (payment_id)
        REFERENCES payments(id)
        ON DELETE SET NULL
);

-- Create indexes for performance
CREATE UNIQUE INDEX idx_blockchain_tx_hash ON blockchain_transactions(tx_hash);
CREATE INDEX idx_blockchain_tx_chain ON blockchain_transactions(chain, network);
CREATE INDEX idx_blockchain_tx_payment_id ON blockchain_transactions(payment_id) WHERE payment_id IS NOT NULL;
CREATE INDEX idx_blockchain_tx_to_address ON blockchain_transactions(to_address);
CREATE INDEX idx_blockchain_tx_from_address ON blockchain_transactions(from_address);
CREATE INDEX idx_blockchain_tx_status ON blockchain_transactions(status);
CREATE INDEX idx_blockchain_tx_is_matched ON blockchain_transactions(is_matched) WHERE is_matched = FALSE;
CREATE INDEX idx_blockchain_tx_created_at ON blockchain_transactions(created_at DESC);
CREATE INDEX idx_blockchain_tx_block_number ON blockchain_transactions(chain, block_number DESC) WHERE block_number IS NOT NULL;
CREATE INDEX idx_blockchain_tx_payment_reference ON blockchain_transactions(parsed_payment_reference) WHERE parsed_payment_reference IS NOT NULL;

-- Composite indexes for common queries
CREATE INDEX idx_blockchain_tx_chain_status ON blockchain_transactions(chain, status, created_at DESC);
CREATE INDEX idx_blockchain_tx_to_address_currency ON blockchain_transactions(to_address, currency, created_at DESC);

-- Add constraints
ALTER TABLE blockchain_transactions ADD CONSTRAINT check_blockchain_chain
    CHECK (chain IN ('solana', 'bsc', 'ethereum'));

ALTER TABLE blockchain_transactions ADD CONSTRAINT check_blockchain_network
    CHECK (network IN ('mainnet', 'testnet', 'devnet'));

ALTER TABLE blockchain_transactions ADD CONSTRAINT check_blockchain_status
    CHECK (status IN ('pending', 'confirmed', 'finalized', 'failed'));

ALTER TABLE blockchain_transactions ADD CONSTRAINT check_blockchain_amount_positive
    CHECK (amount > 0);

ALTER TABLE blockchain_transactions ADD CONSTRAINT check_blockchain_confirmations_non_negative
    CHECK (confirmations >= 0);

-- Create trigger to automatically update updated_at on blockchain_transactions
CREATE TRIGGER update_blockchain_transactions_updated_at
    BEFORE UPDATE ON blockchain_transactions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE blockchain_transactions IS 'Tracks blockchain transaction details for payment confirmation';
COMMENT ON COLUMN blockchain_transactions.chain IS 'Blockchain network: solana, bsc, ethereum';
COMMENT ON COLUMN blockchain_transactions.network IS 'Network type: mainnet, testnet, devnet';
COMMENT ON COLUMN blockchain_transactions.tx_hash IS 'Unique blockchain transaction hash/signature';
COMMENT ON COLUMN blockchain_transactions.memo IS 'Transaction memo field (Solana) or input data (BSC/ETH)';
COMMENT ON COLUMN blockchain_transactions.parsed_payment_reference IS 'Extracted payment reference from memo';
COMMENT ON COLUMN blockchain_transactions.confirmations IS 'Number of block confirmations';
COMMENT ON COLUMN blockchain_transactions.is_finalized IS 'Whether transaction has reached finality';
COMMENT ON COLUMN blockchain_transactions.payment_id IS 'Associated payment ID if matched';
COMMENT ON COLUMN blockchain_transactions.is_matched IS 'Whether transaction has been matched to a payment';
COMMENT ON COLUMN blockchain_transactions.status IS 'Transaction status: pending, confirmed, finalized, failed';
COMMENT ON COLUMN blockchain_transactions.raw_transaction IS 'Full transaction data from blockchain for debugging';
