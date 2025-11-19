-- Migration: Create archival tables (PRD v2.2 - Infinite Data Retention)
-- Purpose: Track archived data in S3 Glacier and maintain transaction hash chain for immutability

-- Table: archived_records - Metadata for data archived to S3 Glacier
CREATE TABLE IF NOT EXISTS archived_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Original Record Information
    original_id UUID NOT NULL, -- Original record ID from source table
    table_name VARCHAR(100) NOT NULL, -- Source table (payments, payouts, ledger_entries, etc.)

    -- Archive Storage Information
    archive_path VARCHAR(500) NOT NULL, -- S3 Glacier object key/path
    s3_bucket VARCHAR(255) NOT NULL,
    s3_region VARCHAR(50) NOT NULL DEFAULT 'ap-southeast-1',
    storage_class VARCHAR(50) DEFAULT 'GLACIER', -- 'GLACIER', 'DEEP_ARCHIVE'

    -- Data Integrity
    data_hash VARCHAR(64) NOT NULL, -- SHA-256 hash of archived data
    compression_algorithm VARCHAR(20) DEFAULT 'gzip', -- Compression used before upload
    encrypted BOOLEAN DEFAULT TRUE, -- Whether data is encrypted in S3
    encryption_algorithm VARCHAR(50) DEFAULT 'AES256',

    -- Archive Size
    original_size_bytes BIGINT, -- Uncompressed size
    compressed_size_bytes BIGINT, -- Compressed size
    object_size_bytes BIGINT, -- Final S3 object size (after compression + encryption)

    -- Archive Lifecycle
    archived_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    archived_by VARCHAR(100), -- System user who triggered archival
    archive_job_id VARCHAR(100), -- Batch job identifier

    -- Restore Information
    restore_initiated_at TIMESTAMP,
    restore_expires_at TIMESTAMP,
    restore_tier VARCHAR(20), -- 'Expedited', 'Standard', 'Bulk'
    restored_at TIMESTAMP,
    restore_initiated_by VARCHAR(100),

    -- Metadata
    record_data_sample JSONB, -- Small sample of original data (for quick reference without restore)
    metadata JSONB DEFAULT '{}', -- Additional flexible metadata

    -- Audit fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for archived_records
CREATE INDEX idx_archived_records_original_id ON archived_records(original_id);
CREATE INDEX idx_archived_records_table_name ON archived_records(table_name);
CREATE INDEX idx_archived_records_data_hash ON archived_records(data_hash);
CREATE INDEX idx_archived_records_archived_at ON archived_records(archived_at DESC);
CREATE INDEX idx_archived_records_archive_path ON archived_records(archive_path);
CREATE UNIQUE INDEX idx_archived_records_unique_record ON archived_records(table_name, original_id);

-- Index for restore operations
CREATE INDEX idx_archived_records_restore_status ON archived_records(restore_initiated_at, restore_expires_at)
    WHERE restore_initiated_at IS NOT NULL;

-- Table: transaction_hashes - SHA-256 hash chain for transaction immutability
CREATE TABLE IF NOT EXISTS transaction_hashes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Hash Chain Information
    table_name VARCHAR(100) NOT NULL, -- Source table
    record_id UUID NOT NULL, -- Original record ID
    data_hash VARCHAR(64) NOT NULL, -- SHA-256 hash of the record data
    previous_hash VARCHAR(64), -- Hash of previous record in chain (NULL for first record)

    -- Merkle Tree
    merkle_root VARCHAR(64), -- Daily Merkle tree root hash (computed at end of day)
    merkle_proof TEXT[], -- Array of hashes forming Merkle proof path

    -- Block Information (like blockchain)
    block_number BIGINT, -- Sequential block number for the day
    block_date DATE NOT NULL, -- Date of the block

    -- Record Snapshot (compressed JSON)
    record_snapshot JSONB, -- Compressed snapshot of record at time of hashing

    -- Verification
    verified BOOLEAN DEFAULT FALSE, -- Whether hash has been verified against record
    verified_at TIMESTAMP,

    -- Audit fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Unique constraint: one hash per record
    CONSTRAINT unique_transaction_hash UNIQUE (table_name, record_id)
);

-- Indexes for transaction_hashes
CREATE UNIQUE INDEX idx_transaction_hashes_record ON transaction_hashes(table_name, record_id);
CREATE INDEX idx_transaction_hashes_data_hash ON transaction_hashes(data_hash);
CREATE INDEX idx_transaction_hashes_block_date ON transaction_hashes(block_date DESC);
CREATE INDEX idx_transaction_hashes_merkle_root ON transaction_hashes(merkle_root) WHERE merkle_root IS NOT NULL;
CREATE INDEX idx_transaction_hashes_created_at ON transaction_hashes(created_at DESC);
CREATE INDEX idx_transaction_hashes_previous_hash ON transaction_hashes(previous_hash) WHERE previous_hash IS NOT NULL;

-- Partial index for unverified hashes
CREATE INDEX idx_transaction_hashes_unverified ON transaction_hashes(table_name, verified) WHERE verified = FALSE;

-- Trigger to update updated_at for archived_records
CREATE OR REPLACE FUNCTION update_archived_records_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_archived_records_updated_at
    BEFORE UPDATE ON archived_records
    FOR EACH ROW
    EXECUTE FUNCTION update_archived_records_updated_at();

-- Trigger to update updated_at for transaction_hashes
CREATE OR REPLACE FUNCTION update_transaction_hashes_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_transaction_hashes_updated_at
    BEFORE UPDATE ON transaction_hashes
    FOR EACH ROW
    EXECUTE FUNCTION update_transaction_hashes_updated_at();

-- Function: Compute SHA-256 hash of JSONB data
CREATE OR REPLACE FUNCTION compute_data_hash(data JSONB)
RETURNS VARCHAR(64) AS $$
BEGIN
    RETURN encode(digest(data::text, 'sha256'), 'hex');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function: Verify hash chain integrity
CREATE OR REPLACE FUNCTION verify_hash_chain(
    p_table_name VARCHAR,
    p_start_date DATE DEFAULT NULL,
    p_end_date DATE DEFAULT NULL
)
RETURNS TABLE(
    total_records BIGINT,
    valid_chain BOOLEAN,
    broken_at UUID,
    error_message TEXT
) AS $$
DECLARE
    prev_hash VARCHAR(64);
    current_record RECORD;
    total_count BIGINT := 0;
    broken_id UUID := NULL;
    error_msg TEXT := NULL;
BEGIN
    -- Get all hashes for the table in chronological order
    FOR current_record IN
        SELECT id, record_id, data_hash, previous_hash, block_date
        FROM transaction_hashes
        WHERE table_name = p_table_name
            AND (p_start_date IS NULL OR block_date >= p_start_date)
            AND (p_end_date IS NULL OR block_date <= p_end_date)
        ORDER BY block_date ASC, created_at ASC
    LOOP
        total_count := total_count + 1;

        -- Check if previous_hash matches actual previous hash
        IF total_count > 1 AND current_record.previous_hash != prev_hash THEN
            broken_id := current_record.id;
            error_msg := format('Hash chain broken at record %s: expected previous_hash %s but got %s',
                current_record.record_id, prev_hash, current_record.previous_hash);
            EXIT;
        END IF;

        prev_hash := current_record.data_hash;
    END LOOP;

    RETURN QUERY SELECT
        total_count,
        (broken_id IS NULL) AS valid_chain,
        broken_id,
        error_msg;
END;
$$ LANGUAGE plpgsql;

-- Check constraints
ALTER TABLE archived_records ADD CONSTRAINT check_archived_storage_class
    CHECK (storage_class IN ('GLACIER', 'DEEP_ARCHIVE', 'INTELLIGENT_TIERING'));

ALTER TABLE archived_records ADD CONSTRAINT check_archived_restore_tier
    CHECK (restore_tier IN ('Expedited', 'Standard', 'Bulk') OR restore_tier IS NULL);

ALTER TABLE archived_records ADD CONSTRAINT check_archived_compression
    CHECK (compression_algorithm IN ('gzip', 'zstd', 'lz4', 'none'));

ALTER TABLE archived_records ADD CONSTRAINT check_archived_size_positive
    CHECK (original_size_bytes > 0 AND compressed_size_bytes > 0 AND object_size_bytes > 0);

-- View: Archive storage statistics
CREATE OR REPLACE VIEW archive_storage_stats AS
SELECT
    table_name,
    COUNT(*) AS total_archived_records,
    SUM(original_size_bytes) AS total_original_bytes,
    SUM(compressed_size_bytes) AS total_compressed_bytes,
    SUM(object_size_bytes) AS total_storage_bytes,
    ROUND(100.0 * SUM(compressed_size_bytes) / NULLIF(SUM(original_size_bytes), 0), 2) AS compression_ratio_pct,
    MIN(archived_at) AS first_archived_at,
    MAX(archived_at) AS last_archived_at,
    COUNT(*) FILTER (WHERE restore_initiated_at IS NOT NULL) AS restore_count,
    COUNT(*) FILTER (WHERE restored_at IS NOT NULL) AS successfully_restored_count
FROM archived_records
GROUP BY table_name;

-- View: Records pending restore (restore initiated but not yet available)
CREATE OR REPLACE VIEW pending_archive_restores AS
SELECT
    id,
    table_name,
    original_id,
    archive_path,
    restore_tier,
    restore_initiated_at,
    restore_expires_at,
    EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - restore_initiated_at)) / 3600 AS hours_since_initiated,
    restore_initiated_by
FROM archived_records
WHERE restore_initiated_at IS NOT NULL
    AND (restored_at IS NULL OR restored_at > CURRENT_TIMESTAMP)
    AND (restore_expires_at IS NULL OR restore_expires_at > CURRENT_TIMESTAMP)
ORDER BY restore_initiated_at ASC;

-- View: Daily hash chain summary
CREATE OR REPLACE VIEW daily_hash_chain_summary AS
SELECT
    block_date,
    table_name,
    COUNT(*) AS record_count,
    COUNT(DISTINCT merkle_root) AS merkle_roots_count,
    MIN(created_at) AS first_hash_at,
    MAX(created_at) AS last_hash_at,
    COUNT(*) FILTER (WHERE verified = TRUE) AS verified_count,
    COUNT(*) FILTER (WHERE verified = FALSE) AS unverified_count
FROM transaction_hashes
GROUP BY block_date, table_name
ORDER BY block_date DESC, table_name;

-- Comments for documentation
COMMENT ON TABLE archived_records IS 'Metadata for records archived to S3 Glacier for infinite data retention (PRD v2.2)';
COMMENT ON TABLE transaction_hashes IS 'SHA-256 hash chain for transaction immutability verification (like blockchain)';
COMMENT ON COLUMN archived_records.data_hash IS 'SHA-256 hash of archived data for integrity verification';
COMMENT ON COLUMN archived_records.restore_tier IS 'Glacier restore speed: Expedited (1-5hr), Standard (3-5hr), Bulk (5-12hr)';
COMMENT ON COLUMN transaction_hashes.previous_hash IS 'Hash of previous record forming chain (NULL for first record of day)';
COMMENT ON COLUMN transaction_hashes.merkle_root IS 'Daily Merkle tree root hash for batch verification';
COMMENT ON FUNCTION compute_data_hash(JSONB) IS 'Compute SHA-256 hash of JSONB data for integrity checking';
COMMENT ON FUNCTION verify_hash_chain(VARCHAR, DATE, DATE) IS 'Verify integrity of hash chain for a table within date range';
COMMENT ON VIEW archive_storage_stats IS 'Storage statistics and compression ratio by table';
COMMENT ON VIEW pending_archive_restores IS 'Archive restore requests that are in progress';
COMMENT ON VIEW daily_hash_chain_summary IS 'Daily summary of hash chain records and verification status';
