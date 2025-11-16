-- Create payouts table
CREATE TABLE IF NOT EXISTS payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL,

    -- Payout amount
    amount_vnd DECIMAL(20, 2) NOT NULL,
    fee_vnd DECIMAL(20, 2) NOT NULL DEFAULT 0,
    net_amount_vnd DECIMAL(20, 2) NOT NULL, -- amount_vnd - fee_vnd

    -- Bank transfer details
    bank_account_name VARCHAR(255) NOT NULL,
    bank_account_number VARCHAR(50) NOT NULL,
    bank_name VARCHAR(100) NOT NULL,
    bank_branch VARCHAR(100),

    -- Status: requested, approved, processing, completed, rejected, failed
    status VARCHAR(20) NOT NULL DEFAULT 'requested',

    -- Approval workflow
    requested_by UUID NOT NULL, -- merchant_id
    approved_by UUID, -- admin user id
    approved_at TIMESTAMP,
    rejection_reason TEXT,

    -- Processing details
    processed_by UUID, -- ops user id who executed transfer
    processed_at TIMESTAMP,
    completion_date TIMESTAMP,

    -- Bank transfer reference
    bank_reference_number VARCHAR(100),
    transaction_receipt_url TEXT,

    -- Failure tracking
    failure_reason TEXT,
    retry_count INT DEFAULT 0,

    -- Metadata
    notes TEXT, -- Internal notes from ops team
    metadata JSONB,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    -- Foreign key constraint
    CONSTRAINT fk_payouts_merchant
        FOREIGN KEY (merchant_id)
        REFERENCES merchants(id)
        ON DELETE RESTRICT
);

-- Create indexes for performance
CREATE INDEX idx_payouts_merchant_id ON payouts(merchant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payouts_status ON payouts(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_payouts_merchant_status ON payouts(merchant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_payouts_created_at ON payouts(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_payouts_approved_at ON payouts(approved_at) WHERE deleted_at IS NULL AND status IN ('approved', 'processing', 'completed');

-- Add constraints to ensure payout status is valid
ALTER TABLE payouts ADD CONSTRAINT check_payout_status
    CHECK (status IN ('requested', 'approved', 'processing', 'completed', 'rejected', 'failed'));

-- Add constraint to ensure amounts are positive
ALTER TABLE payouts ADD CONSTRAINT check_payout_amounts_positive
    CHECK (amount_vnd > 0 AND fee_vnd >= 0 AND net_amount_vnd > 0);

-- Add constraint to ensure net amount is correct
ALTER TABLE payouts ADD CONSTRAINT check_payout_net_amount
    CHECK (net_amount_vnd = amount_vnd - fee_vnd);

-- Create trigger to automatically update updated_at on payouts
CREATE TRIGGER update_payouts_updated_at
    BEFORE UPDATE ON payouts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE payouts IS 'Records merchant withdrawal requests and their processing status';
COMMENT ON COLUMN payouts.status IS 'Payout status: requested → approved → processing → completed | rejected | failed';
COMMENT ON COLUMN payouts.amount_vnd IS 'Total payout amount requested by merchant';
COMMENT ON COLUMN payouts.fee_vnd IS 'Payout processing fee charged to merchant';
COMMENT ON COLUMN payouts.net_amount_vnd IS 'Actual amount transferred to merchant bank account';
COMMENT ON COLUMN payouts.bank_reference_number IS 'Bank transaction reference number for reconciliation';
COMMENT ON COLUMN payouts.approved_by IS 'Admin user who approved the payout request';
COMMENT ON COLUMN payouts.processed_by IS 'Ops user who executed the bank transfer';
