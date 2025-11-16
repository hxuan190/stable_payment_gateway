-- Create merchant_balances table
-- This table maintains current balance state for quick lookups
-- Balances are computed from ledger_entries and cached here for performance
CREATE TABLE IF NOT EXISTS merchant_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL UNIQUE,

    -- VND balances
    pending_vnd DECIMAL(20, 2) NOT NULL DEFAULT 0,
    available_vnd DECIMAL(20, 2) NOT NULL DEFAULT 0,
    total_vnd DECIMAL(20, 2) NOT NULL DEFAULT 0, -- pending + available

    -- Reserved/locked balance (for pending payouts)
    reserved_vnd DECIMAL(20, 2) NOT NULL DEFAULT 0,

    -- Lifetime statistics
    total_received_vnd DECIMAL(20, 2) NOT NULL DEFAULT 0,
    total_paid_out_vnd DECIMAL(20, 2) NOT NULL DEFAULT 0,
    total_fees_vnd DECIMAL(20, 2) NOT NULL DEFAULT 0,

    -- Transaction counts
    total_payments_count INT NOT NULL DEFAULT 0,
    total_payouts_count INT NOT NULL DEFAULT 0,

    -- Last transaction timestamps
    last_payment_at TIMESTAMP,
    last_payout_at TIMESTAMP,

    -- Version for optimistic locking to prevent concurrent update conflicts
    version INT NOT NULL DEFAULT 1,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Foreign key constraint
    CONSTRAINT fk_merchant_balances_merchant
        FOREIGN KEY (merchant_id)
        REFERENCES merchants(id)
        ON DELETE RESTRICT
);

-- Create index on merchant_id
CREATE INDEX idx_merchant_balances_merchant_id ON merchant_balances(merchant_id);

-- Add constraints to ensure balances are non-negative
ALTER TABLE merchant_balances ADD CONSTRAINT check_merchant_balances_non_negative
    CHECK (
        pending_vnd >= 0 AND
        available_vnd >= 0 AND
        total_vnd >= 0 AND
        reserved_vnd >= 0 AND
        total_received_vnd >= 0 AND
        total_paid_out_vnd >= 0 AND
        total_fees_vnd >= 0
    );

-- Add constraint to ensure total = pending + available
ALTER TABLE merchant_balances ADD CONSTRAINT check_merchant_balances_total
    CHECK (total_vnd = pending_vnd + available_vnd);

-- Create trigger to automatically update updated_at on merchant_balances
CREATE TRIGGER update_merchant_balances_updated_at
    BEFORE UPDATE ON merchant_balances
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create function to automatically create balance record when merchant is created
CREATE OR REPLACE FUNCTION create_merchant_balance()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO merchant_balances (merchant_id)
    VALUES (NEW.id);
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to automatically create balance when merchant is created
CREATE TRIGGER create_merchant_balance_on_merchant_insert
    AFTER INSERT ON merchants
    FOR EACH ROW
    EXECUTE FUNCTION create_merchant_balance();

-- Add comments for documentation
COMMENT ON TABLE merchant_balances IS 'Cached merchant balance state for quick lookups (source of truth is ledger_entries)';
COMMENT ON COLUMN merchant_balances.pending_vnd IS 'Balance pending OTC conversion';
COMMENT ON COLUMN merchant_balances.available_vnd IS 'Balance available for payout';
COMMENT ON COLUMN merchant_balances.reserved_vnd IS 'Balance locked for pending payout requests';
COMMENT ON COLUMN merchant_balances.total_vnd IS 'Total balance (pending + available)';
COMMENT ON COLUMN merchant_balances.version IS 'Optimistic locking version to prevent concurrent update conflicts';
COMMENT ON COLUMN merchant_balances.total_received_vnd IS 'Lifetime total payments received';
COMMENT ON COLUMN merchant_balances.total_paid_out_vnd IS 'Lifetime total payouts processed';
COMMENT ON COLUMN merchant_balances.total_fees_vnd IS 'Lifetime total fees charged';
