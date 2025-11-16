-- Drop view
DROP VIEW IF EXISTS ledger_balances;

-- Drop indexes
DROP INDEX IF EXISTS idx_ledger_entries_merchant_currency;
DROP INDEX IF EXISTS idx_ledger_entries_currency;
DROP INDEX IF EXISTS idx_ledger_entries_transaction_group;
DROP INDEX IF EXISTS idx_ledger_entries_created_at;
DROP INDEX IF EXISTS idx_ledger_entries_credit_account;
DROP INDEX IF EXISTS idx_ledger_entries_debit_account;
DROP INDEX IF EXISTS idx_ledger_entries_reference;
DROP INDEX IF EXISTS idx_ledger_entries_merchant_id;

-- Drop table
DROP TABLE IF EXISTS ledger_entries;
