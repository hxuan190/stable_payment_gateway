-- Drop kyc_documents table and related objects
DROP TRIGGER IF EXISTS update_kyc_documents_updated_at ON kyc_documents;

DROP INDEX IF EXISTS idx_kyc_documents_merchant_status;
DROP INDEX IF EXISTS idx_kyc_documents_created_at;
DROP INDEX IF EXISTS idx_kyc_documents_status;
DROP INDEX IF EXISTS idx_kyc_documents_merchant;

DROP TABLE IF EXISTS kyc_documents;
