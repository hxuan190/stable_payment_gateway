-- Create travel_rule_data table for FATF compliance
-- Required for transactions > $1000 USD equivalent
CREATE TABLE IF NOT EXISTS travel_rule_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL,

    -- Payer Information (Originator)
    payer_full_name VARCHAR(255) NOT NULL,
    payer_wallet_address VARCHAR(255) NOT NULL,
    payer_country CHAR(2) NOT NULL, -- ISO 3166-1 alpha-2 country code
    payer_id_document VARCHAR(255), -- Optional: passport, national ID, etc.

    -- Merchant Information (Beneficiary)
    merchant_full_name VARCHAR(255) NOT NULL,
    merchant_country CHAR(2) NOT NULL, -- ISO 3166-1 alpha-2 country code

    -- Transaction Information
    transaction_amount DECIMAL(20, 8) NOT NULL, -- Amount in USD equivalent
    transaction_currency VARCHAR(10) NOT NULL, -- USD, USDT, USDC, BUSD

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Foreign key constraint to payments table
    CONSTRAINT fk_travel_rule_payment
        FOREIGN KEY (payment_id)
        REFERENCES payments(id)
        ON DELETE CASCADE
);

-- Create indexes for performance
CREATE INDEX idx_travel_rule_payment_id ON travel_rule_data(payment_id);
CREATE INDEX idx_travel_rule_created_at ON travel_rule_data(created_at DESC);
CREATE INDEX idx_travel_rule_payer_country ON travel_rule_data(payer_country);
CREATE INDEX idx_travel_rule_merchant_country ON travel_rule_data(merchant_country);
CREATE INDEX idx_travel_rule_payer_wallet ON travel_rule_data(payer_wallet_address);

-- Add constraints to ensure valid data
ALTER TABLE travel_rule_data ADD CONSTRAINT check_travel_rule_amount_positive
    CHECK (transaction_amount > 0);

ALTER TABLE travel_rule_data ADD CONSTRAINT check_travel_rule_country_code_length
    CHECK (LENGTH(payer_country) = 2 AND LENGTH(merchant_country) = 2);

ALTER TABLE travel_rule_data ADD CONSTRAINT check_travel_rule_currency
    CHECK (transaction_currency IN ('USD', 'USDT', 'USDC', 'BUSD'));

-- Add comments for documentation
COMMENT ON TABLE travel_rule_data IS 'Stores Travel Rule compliance data for transactions > $1000 USD per FATF recommendations';
COMMENT ON COLUMN travel_rule_data.payment_id IS 'Reference to the payment this Travel Rule data belongs to';
COMMENT ON COLUMN travel_rule_data.payer_full_name IS 'Full legal name of the person making the payment (originator)';
COMMENT ON COLUMN travel_rule_data.payer_wallet_address IS 'Blockchain wallet address of the payer';
COMMENT ON COLUMN travel_rule_data.payer_country IS 'ISO 3166-1 alpha-2 country code of payer (e.g., VN, US, SG)';
COMMENT ON COLUMN travel_rule_data.payer_id_document IS 'Optional ID document number for enhanced due diligence';
COMMENT ON COLUMN travel_rule_data.merchant_full_name IS 'Full legal name of the merchant receiving payment (beneficiary)';
COMMENT ON COLUMN travel_rule_data.merchant_country IS 'ISO 3166-1 alpha-2 country code of merchant';
COMMENT ON COLUMN travel_rule_data.transaction_amount IS 'Transaction amount in USD equivalent for threshold calculation';
COMMENT ON COLUMN travel_rule_data.transaction_currency IS 'Currency used for the transaction';
