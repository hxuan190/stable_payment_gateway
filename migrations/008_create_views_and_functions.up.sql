-- Create helpful views and functions for common queries

-- View: Active merchants with their current balance
CREATE OR REPLACE VIEW active_merchants_with_balance AS
SELECT
    m.id,
    m.email,
    m.business_name,
    m.kyc_status,
    m.status,
    m.created_at AS merchant_since,
    b.available_vnd,
    b.pending_vnd,
    b.total_vnd,
    b.reserved_vnd,
    b.total_received_vnd,
    b.total_paid_out_vnd,
    b.total_fees_vnd,
    b.total_payments_count,
    b.total_payouts_count,
    b.last_payment_at,
    b.last_payout_at
FROM merchants m
LEFT JOIN merchant_balances b ON m.id = b.merchant_id
WHERE m.deleted_at IS NULL
    AND m.status = 'active';

COMMENT ON VIEW active_merchants_with_balance IS 'Active merchants with their current balance information';

-- View: Recent payments summary
CREATE OR REPLACE VIEW recent_payments_summary AS
SELECT
    p.id,
    p.merchant_id,
    m.business_name AS merchant_name,
    p.amount_vnd,
    p.amount_crypto,
    p.currency,
    p.chain,
    p.status,
    p.order_id,
    p.tx_hash,
    p.created_at,
    p.confirmed_at,
    EXTRACT(EPOCH FROM (p.confirmed_at - p.created_at)) AS confirmation_time_seconds
FROM payments p
JOIN merchants m ON p.merchant_id = m.id
WHERE p.deleted_at IS NULL
ORDER BY p.created_at DESC;

COMMENT ON VIEW recent_payments_summary IS 'Recent payments with merchant details and confirmation times';

-- View: Pending payouts for admin review
CREATE OR REPLACE VIEW pending_payouts_for_review AS
SELECT
    po.id,
    po.merchant_id,
    m.business_name AS merchant_name,
    m.email AS merchant_email,
    po.amount_vnd,
    po.fee_vnd,
    po.net_amount_vnd,
    po.bank_account_name,
    po.bank_account_number,
    po.bank_name,
    po.status,
    po.created_at AS requested_at,
    b.available_vnd AS merchant_available_balance,
    b.total_vnd AS merchant_total_balance
FROM payouts po
JOIN merchants m ON po.merchant_id = m.id
LEFT JOIN merchant_balances b ON po.merchant_id = b.merchant_id
WHERE po.deleted_at IS NULL
    AND po.status IN ('requested', 'approved')
ORDER BY po.created_at ASC;

COMMENT ON VIEW pending_payouts_for_review IS 'Pending payout requests with merchant balance information';

-- View: System statistics dashboard
CREATE OR REPLACE VIEW system_statistics AS
SELECT
    (SELECT COUNT(*) FROM merchants WHERE deleted_at IS NULL AND status = 'active') AS active_merchants_count,
    (SELECT COUNT(*) FROM merchants WHERE deleted_at IS NULL AND kyc_status = 'pending') AS pending_kyc_count,
    (SELECT COUNT(*) FROM payments WHERE deleted_at IS NULL AND status = 'completed' AND DATE(created_at) = CURRENT_DATE) AS payments_today_count,
    (SELECT COALESCE(SUM(amount_vnd), 0) FROM payments WHERE deleted_at IS NULL AND status = 'completed' AND DATE(created_at) = CURRENT_DATE) AS volume_today_vnd,
    (SELECT COUNT(*) FROM payments WHERE deleted_at IS NULL AND status IN ('created', 'pending', 'confirming')) AS pending_payments_count,
    (SELECT COUNT(*) FROM payouts WHERE deleted_at IS NULL AND status = 'requested') AS pending_payouts_count,
    (SELECT COALESCE(SUM(available_vnd), 0) FROM merchant_balances) AS total_merchant_available_vnd,
    (SELECT COALESCE(SUM(total_vnd), 0) FROM merchant_balances) AS total_merchant_balance_vnd,
    (SELECT COALESCE(SUM(fee_vnd), 0) FROM payments WHERE deleted_at IS NULL AND status = 'completed' AND DATE(created_at) >= CURRENT_DATE - INTERVAL '30 days') AS fees_last_30_days_vnd,
    (SELECT COUNT(*) FROM payments WHERE deleted_at IS NULL AND status = 'completed' AND DATE(created_at) >= CURRENT_DATE - INTERVAL '30 days') AS payments_last_30_days_count,
    (SELECT COALESCE(SUM(amount_vnd), 0) FROM payments WHERE deleted_at IS NULL AND status = 'completed' AND DATE(created_at) >= CURRENT_DATE - INTERVAL '30 days') AS volume_last_30_days_vnd;

COMMENT ON VIEW system_statistics IS 'System-wide statistics for admin dashboard';

-- View: Unmatched blockchain transactions
CREATE OR REPLACE VIEW unmatched_blockchain_transactions AS
SELECT
    bt.id,
    bt.chain,
    bt.tx_hash,
    bt.from_address,
    bt.to_address,
    bt.amount,
    bt.currency,
    bt.memo,
    bt.parsed_payment_reference,
    bt.status,
    bt.is_finalized,
    bt.created_at,
    EXTRACT(EPOCH FROM (NOW() - bt.created_at))/60 AS minutes_since_detected
FROM blockchain_transactions bt
WHERE bt.is_matched = FALSE
    AND bt.status IN ('confirmed', 'finalized')
ORDER BY bt.created_at DESC;

COMMENT ON VIEW unmatched_blockchain_transactions IS 'Blockchain transactions that have not been matched to payments';

-- Function: Calculate merchant balance from ledger entries
CREATE OR REPLACE FUNCTION calculate_merchant_balance_from_ledger(p_merchant_id UUID, p_currency VARCHAR DEFAULT 'VND')
RETURNS TABLE (
    pending_balance DECIMAL,
    available_balance DECIMAL,
    total_balance DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COALESCE(SUM(CASE
            WHEN credit_account LIKE 'merchant:' || p_merchant_id || ':pending' THEN amount
            WHEN debit_account LIKE 'merchant:' || p_merchant_id || ':pending' THEN -amount
            ELSE 0
        END), 0) AS pending_balance,
        COALESCE(SUM(CASE
            WHEN credit_account LIKE 'merchant:' || p_merchant_id || ':available' THEN amount
            WHEN debit_account LIKE 'merchant:' || p_merchant_id || ':available' THEN -amount
            ELSE 0
        END), 0) AS available_balance,
        COALESCE(SUM(CASE
            WHEN credit_account LIKE 'merchant:' || p_merchant_id || '%' THEN amount
            WHEN debit_account LIKE 'merchant:' || p_merchant_id || '%' THEN -amount
            ELSE 0
        END), 0) AS total_balance
    FROM ledger_entries
    WHERE merchant_id = p_merchant_id
        AND currency = p_currency;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION calculate_merchant_balance_from_ledger IS 'Calculates merchant balance from ledger entries (for reconciliation)';

-- Function: Get payment statistics for merchant
CREATE OR REPLACE FUNCTION get_merchant_payment_stats(p_merchant_id UUID, p_days INT DEFAULT 30)
RETURNS TABLE (
    total_payments BIGINT,
    completed_payments BIGINT,
    failed_payments BIGINT,
    total_volume_vnd DECIMAL,
    total_fees_vnd DECIMAL,
    avg_payment_vnd DECIMAL,
    avg_confirmation_time_seconds DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::BIGINT AS total_payments,
        COUNT(*) FILTER (WHERE status = 'completed')::BIGINT AS completed_payments,
        COUNT(*) FILTER (WHERE status IN ('failed', 'expired'))::BIGINT AS failed_payments,
        COALESCE(SUM(amount_vnd) FILTER (WHERE status = 'completed'), 0) AS total_volume_vnd,
        COALESCE(SUM(fee_vnd) FILTER (WHERE status = 'completed'), 0) AS total_fees_vnd,
        COALESCE(AVG(amount_vnd) FILTER (WHERE status = 'completed'), 0) AS avg_payment_vnd,
        COALESCE(AVG(EXTRACT(EPOCH FROM (confirmed_at - created_at))) FILTER (WHERE status = 'completed' AND confirmed_at IS NOT NULL), 0) AS avg_confirmation_time_seconds
    FROM payments
    WHERE merchant_id = p_merchant_id
        AND deleted_at IS NULL
        AND created_at >= CURRENT_DATE - INTERVAL '1 day' * p_days;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_merchant_payment_stats IS 'Get payment statistics for a merchant over specified time period';

-- Create extension for UUID generation (if not already exists)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create extension for pg_trgm for fuzzy text search (useful for merchant search)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Create GIN index for full-text search on merchant business names
CREATE INDEX idx_merchants_business_name_trgm ON merchants USING gin (business_name gin_trgm_ops);

-- Create GIN index for JSONB metadata fields for faster queries
CREATE INDEX idx_payments_metadata_gin ON payments USING gin (metadata) WHERE metadata IS NOT NULL;
CREATE INDEX idx_merchants_metadata_gin ON merchants USING gin (metadata) WHERE metadata IS NOT NULL;
CREATE INDEX idx_audit_logs_metadata_gin ON audit_logs USING gin (metadata) WHERE metadata IS NOT NULL;
