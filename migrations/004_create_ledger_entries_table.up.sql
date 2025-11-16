-- Create ledger_entries table for double-entry accounting
CREATE TABLE IF NOT EXISTS ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Double-entry accounting: every transaction has debit and credit
    -- Account types:
    -- - crypto_pool: System's crypto holdings
    -- - vnd_pool: System's VND holdings
    -- - merchant_pending_balance: Merchant balance pending conversion
    -- - merchant_available_balance: Merchant balance available for payout
    -- - fee_revenue: Platform fee revenue
    -- - merchant:{merchant_id}:pending: Merchant-specific pending balance
    -- - merchant:{merchant_id}:available: Merchant-specific available balance

    debit_account VARCHAR(255) NOT NULL,
    credit_account VARCHAR(255) NOT NULL,

    -- Amount and currency
    amount DECIMAL(20, 8) NOT NULL,
    currency VARCHAR(10) NOT NULL, -- VND, USDT, USDC, etc.

    -- Reference to the source transaction
    reference_type VARCHAR(50) NOT NULL, -- payment, payout, otc_conversion, fee, refund, adjustment
    reference_id UUID NOT NULL, -- ID of the payment, payout, etc.

    -- Associated merchant (if applicable)
    merchant_id UUID,

    -- Transaction description
    description TEXT NOT NULL,

    -- Entry pair ID (links debit and credit entries that are part of same transaction)
    transaction_group UUID NOT NULL,

    -- Entry type: debit or credit
    entry_type VARCHAR(10) NOT NULL,

    -- Metadata for additional context
    metadata JSONB,

    -- Timestamps - ledger entries are immutable
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Ledger entries should never be updated or deleted (append-only)
    -- No updated_at or deleted_at columns

    -- Foreign key constraint (optional, merchant can be null for system accounts)
    CONSTRAINT fk_ledger_entries_merchant
        FOREIGN KEY (merchant_id)
        REFERENCES merchants(id)
        ON DELETE RESTRICT
);

-- Create indexes for performance
CREATE INDEX idx_ledger_entries_merchant_id ON ledger_entries(merchant_id) WHERE merchant_id IS NOT NULL;
CREATE INDEX idx_ledger_entries_reference ON ledger_entries(reference_type, reference_id);
CREATE INDEX idx_ledger_entries_debit_account ON ledger_entries(debit_account);
CREATE INDEX idx_ledger_entries_credit_account ON ledger_entries(credit_account);
CREATE INDEX idx_ledger_entries_created_at ON ledger_entries(created_at DESC);
CREATE INDEX idx_ledger_entries_transaction_group ON ledger_entries(transaction_group);
CREATE INDEX idx_ledger_entries_currency ON ledger_entries(currency);

-- Composite index for merchant balance queries
CREATE INDEX idx_ledger_entries_merchant_currency ON ledger_entries(merchant_id, currency, created_at DESC)
    WHERE merchant_id IS NOT NULL;

-- Add constraints to ensure entry type is valid
ALTER TABLE ledger_entries ADD CONSTRAINT check_ledger_entry_type
    CHECK (entry_type IN ('debit', 'credit'));

-- Add constraint to ensure reference type is valid
ALTER TABLE ledger_entries ADD CONSTRAINT check_ledger_reference_type
    CHECK (reference_type IN ('payment', 'payout', 'otc_conversion', 'fee', 'refund', 'adjustment'));

-- Add constraint to ensure amount is positive
ALTER TABLE ledger_entries ADD CONSTRAINT check_ledger_amount_positive
    CHECK (amount > 0);

-- Add constraint to ensure debit and credit accounts are different
ALTER TABLE ledger_entries ADD CONSTRAINT check_ledger_accounts_different
    CHECK (debit_account != credit_account);

-- Add comments for documentation
COMMENT ON TABLE ledger_entries IS 'Double-entry accounting ledger (append-only, immutable)';
COMMENT ON COLUMN ledger_entries.debit_account IS 'Account to debit (money leaving this account)';
COMMENT ON COLUMN ledger_entries.credit_account IS 'Account to credit (money entering this account)';
COMMENT ON COLUMN ledger_entries.transaction_group IS 'Groups related debit/credit entries together';
COMMENT ON COLUMN ledger_entries.reference_type IS 'Type of transaction: payment, payout, otc_conversion, fee, refund, adjustment';
COMMENT ON COLUMN ledger_entries.reference_id IS 'ID of the referenced transaction (payment_id, payout_id, etc.)';
COMMENT ON COLUMN ledger_entries.entry_type IS 'Entry type: debit (outflow) or credit (inflow)';
COMMENT ON COLUMN ledger_entries.merchant_id IS 'Associated merchant (NULL for system accounts)';

-- Create view for balance calculation helper
CREATE OR REPLACE VIEW ledger_balances AS
SELECT
    merchant_id,
    currency,
    debit_account AS account,
    SUM(CASE WHEN entry_type = 'debit' THEN -amount ELSE amount END) AS balance,
    COUNT(*) AS entry_count,
    MAX(created_at) AS last_updated
FROM ledger_entries
WHERE merchant_id IS NOT NULL
GROUP BY merchant_id, currency, debit_account

UNION ALL

SELECT
    merchant_id,
    currency,
    credit_account AS account,
    SUM(CASE WHEN entry_type = 'credit' THEN amount ELSE -amount END) AS balance,
    COUNT(*) AS entry_count,
    MAX(created_at) AS last_updated
FROM ledger_entries
WHERE merchant_id IS NOT NULL
GROUP BY merchant_id, currency, credit_account;

COMMENT ON VIEW ledger_balances IS 'Helper view for calculating account balances from ledger entries';
