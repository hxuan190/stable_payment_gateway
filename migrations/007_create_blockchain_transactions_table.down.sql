-- Drop trigger
DROP TRIGGER IF EXISTS update_blockchain_transactions_updated_at ON blockchain_transactions;

-- Drop indexes
DROP INDEX IF EXISTS idx_blockchain_tx_to_address_currency;
DROP INDEX IF EXISTS idx_blockchain_tx_chain_status;
DROP INDEX IF EXISTS idx_blockchain_tx_payment_reference;
DROP INDEX IF EXISTS idx_blockchain_tx_block_number;
DROP INDEX IF EXISTS idx_blockchain_tx_created_at;
DROP INDEX IF EXISTS idx_blockchain_tx_is_matched;
DROP INDEX IF EXISTS idx_blockchain_tx_status;
DROP INDEX IF EXISTS idx_blockchain_tx_from_address;
DROP INDEX IF EXISTS idx_blockchain_tx_to_address;
DROP INDEX IF EXISTS idx_blockchain_tx_payment_id;
DROP INDEX IF EXISTS idx_blockchain_tx_chain;
DROP INDEX IF EXISTS idx_blockchain_tx_hash;

-- Drop table
DROP TABLE IF EXISTS blockchain_transactions;
