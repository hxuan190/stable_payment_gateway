-- Create kyc_documents table for storing merchant KYC document metadata
-- Actual files stored in S3 with encryption

CREATE TABLE IF NOT EXISTS kyc_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL,

    -- Document information
    document_type VARCHAR(50) NOT NULL, -- business_registration, tax_certificate, owner_id, bank_statement, etc.
    file_url TEXT NOT NULL, -- S3 URL
    file_size_bytes BIGINT,
    mime_type VARCHAR(100), -- application/pdf, image/jpeg, etc.

    -- Review status
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, approved, rejected
    reviewed_by UUID, -- Admin user who reviewed
    reviewed_at TIMESTAMP,
    reviewer_notes TEXT,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Foreign key constraint to merchants table
    CONSTRAINT fk_kyc_documents_merchant
        FOREIGN KEY (merchant_id)
        REFERENCES merchants(id)
        ON DELETE CASCADE
);

-- Create indexes for performance
CREATE INDEX idx_kyc_documents_merchant ON kyc_documents(merchant_id);
CREATE INDEX idx_kyc_documents_status ON kyc_documents(status);
CREATE INDEX idx_kyc_documents_created_at ON kyc_documents(created_at DESC);
CREATE INDEX idx_kyc_documents_merchant_status ON kyc_documents(merchant_id, status);

-- Add constraints to ensure valid data
ALTER TABLE kyc_documents ADD CONSTRAINT check_kyc_documents_status
    CHECK (status IN ('pending', 'approved', 'rejected'));

ALTER TABLE kyc_documents ADD CONSTRAINT check_kyc_documents_file_size
    CHECK (file_size_bytes > 0 AND file_size_bytes <= 10485760); -- Max 10MB

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_kyc_documents_updated_at
    BEFORE UPDATE ON kyc_documents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE kyc_documents IS 'Stores metadata for KYC documents uploaded by merchants (files stored in S3)';
COMMENT ON COLUMN kyc_documents.document_type IS 'Type of document: business_registration, tax_certificate, owner_id, bank_statement, etc.';
COMMENT ON COLUMN kyc_documents.file_url IS 'S3 URL where the encrypted document is stored';
COMMENT ON COLUMN kyc_documents.status IS 'Review status: pending, approved, rejected';
COMMENT ON COLUMN kyc_documents.reviewed_by IS 'UUID of admin user who reviewed the document';
COMMENT ON COLUMN kyc_documents.reviewer_notes IS 'Notes from admin reviewer (especially for rejections)';
