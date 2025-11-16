-- Create payments table
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL,

    -- Payment details
    amount_vnd DECIMAL(20, 2) NOT NULL,
    amount_crypto DECIMAL(20, 8) NOT NULL,
    currency VARCHAR(10) NOT NULL, -- USDT, USDC, etc.
    chain VARCHAR(20) NOT NULL, -- solana, bsc, ethereum
    exchange_rate DECIMAL(20, 6) NOT NULL, -- VND per crypto at time of creation

    -- Merchant reference
    order_id VARCHAR(255), -- Merchant's order reference
    description TEXT,
    callback_url TEXT, -- Merchant's webhook URL for this specific payment (optional override)

    -- Payment status: created, pending, confirming, completed, expired, failed
    status VARCHAR(20) NOT NULL DEFAULT 'created',

    -- Blockchain transaction details
    tx_hash VARCHAR(255),
    tx_confirmations INT DEFAULT 0,
    from_address VARCHAR(255),

    -- Payment memo/reference (used to match blockchain transaction to payment)
    payment_reference VARCHAR(100) NOT NULL UNIQUE,

    -- Wallet address where payment should be sent
    destination_wallet VARCHAR(255) NOT NULL,

    -- Timing
    expires_at TIMESTAMP NOT NULL,
    paid_at TIMESTAMP,
    confirmed_at TIMESTAMP,

    -- Fee calculation (1% default)
    fee_percentage DECIMAL(5, 4) NOT NULL DEFAULT 0.0100, -- 1.00%
    fee_vnd DECIMAL(20, 2),
    net_amount_vnd DECIMAL(20, 2), -- Amount after fee

    -- Status tracking
    failure_reason TEXT,

    -- Metadata
    metadata JSONB,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    -- Foreign key constraint
    CONSTRAINT fk_payments_merchant
        FOREIGN KEY (merchant_id)
        REFERENCES merchants(id)
        ON DELETE RESTRICT
);

-- Create indexes for performance
CREATE INDEX idx_payments_merchant_id ON payments(merchant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payments_status ON payments(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_payments_tx_hash ON payments(tx_hash) WHERE deleted_at IS NULL AND tx_hash IS NOT NULL;
CREATE INDEX idx_payments_payment_reference ON payments(payment_reference) WHERE deleted_at IS NULL;
CREATE INDEX idx_payments_order_id ON payments(order_id) WHERE deleted_at IS NULL AND order_id IS NOT NULL;
CREATE INDEX idx_payments_created_at ON payments(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_payments_expires_at ON payments(expires_at) WHERE deleted_at IS NULL AND status IN ('created', 'pending');
CREATE INDEX idx_payments_merchant_status ON payments(merchant_id, status) WHERE deleted_at IS NULL;

-- Add constraints to ensure payment status is valid
ALTER TABLE payments ADD CONSTRAINT check_payment_status
    CHECK (status IN ('created', 'pending', 'confirming', 'completed', 'expired', 'failed'));

-- Add constraint to ensure chain is valid
ALTER TABLE payments ADD CONSTRAINT check_payment_chain
    CHECK (chain IN ('solana', 'bsc', 'ethereum'));

-- Add constraint to ensure amounts are positive
ALTER TABLE payments ADD CONSTRAINT check_payment_amounts_positive
    CHECK (amount_vnd > 0 AND amount_crypto > 0);

-- Add constraint to ensure fee percentage is valid (0-10%)
ALTER TABLE payments ADD CONSTRAINT check_payment_fee_percentage
    CHECK (fee_percentage >= 0 AND fee_percentage <= 0.1000);

-- Create trigger to automatically update updated_at on payments
CREATE TRIGGER update_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE payments IS 'Stores all payment requests and their lifecycle';
COMMENT ON COLUMN payments.status IS 'Payment status: created → pending → confirming → completed | expired | failed';
COMMENT ON COLUMN payments.payment_reference IS 'Unique reference used in blockchain memo to match transaction';
COMMENT ON COLUMN payments.amount_vnd IS 'Original payment amount requested in VND';
COMMENT ON COLUMN payments.amount_crypto IS 'Equivalent amount in cryptocurrency (USDT/USDC)';
COMMENT ON COLUMN payments.exchange_rate IS 'Exchange rate at time of payment creation (VND per crypto unit)';
COMMENT ON COLUMN payments.tx_hash IS 'Blockchain transaction hash once payment is made';
COMMENT ON COLUMN payments.fee_vnd IS 'Platform fee charged on this payment';
COMMENT ON COLUMN payments.net_amount_vnd IS 'Net amount credited to merchant after fee';
