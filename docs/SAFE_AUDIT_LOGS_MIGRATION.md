# Safe Audit Logs Partitioning Migration

**Risk Level**: 🔴 CRITICAL
**Downtime**: Zero (with careful execution)
**Reversible**: Yes (within 24 hours)

---

## 🎯 Migration Strategy

### Strategy: "Shadow Table" Approach

Instead of altering the live table, we:
1. Create partitioned table alongside existing table
2. Dual-write to both tables (application change)
3. Backfill old data gradually
4. Verify data integrity
5. Switch reads to new table
6. Drop old table after confidence period

---

## 📋 Step-by-Step Execution Plan

### **Phase 1: Preparation** (Week -1)

#### Step 1.1: Create Migration Script
```sql
-- migrations/013_create_audit_logs_partitioned.up.sql

-- Create partitioned table with new name (not replacing yet)
CREATE TABLE audit_logs_v2 (
    id UUID NOT NULL,
    actor_type VARCHAR(50) NOT NULL,
    actor_id UUID,
    actor_email VARCHAR(255),
    actor_ip_address VARCHAR(45),
    action VARCHAR(100) NOT NULL,
    action_category VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    error_message TEXT,
    http_method VARCHAR(10),
    http_path TEXT,
    http_status_code INT,
    user_agent TEXT,
    old_values JSONB,
    new_values JSONB,
    metadata JSONB,
    description TEXT,
    request_id UUID,
    correlation_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Create partitions for 5 years (2025-2030)
CREATE TABLE audit_logs_v2_2025 PARTITION OF audit_logs_v2
    FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');

CREATE TABLE audit_logs_v2_2026 PARTITION OF audit_logs_v2
    FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');

CREATE TABLE audit_logs_v2_2027 PARTITION OF audit_logs_v2
    FOR VALUES FROM ('2027-01-01') TO ('2028-01-01');

CREATE TABLE audit_logs_v2_2028 PARTITION OF audit_logs_v2
    FOR VALUES FROM ('2028-01-01') TO ('2029-01-01');

CREATE TABLE audit_logs_v2_2029 PARTITION OF audit_logs_v2
    FOR VALUES FROM ('2029-01-01') TO ('2030-01-01');

CREATE TABLE audit_logs_v2_2030 PARTITION OF audit_logs_v2
    FOR VALUES FROM ('2030-01-01') TO ('2031-01-01');

-- Create indexes (on partitioned table, applies to all partitions)
CREATE INDEX idx_audit_logs_v2_actor_id ON audit_logs_v2(actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_audit_logs_v2_actor_type ON audit_logs_v2(actor_type);
CREATE INDEX idx_audit_logs_v2_action ON audit_logs_v2(action);
CREATE INDEX idx_audit_logs_v2_action_category ON audit_logs_v2(action_category);
CREATE INDEX idx_audit_logs_v2_resource ON audit_logs_v2(resource_type, resource_id);
CREATE INDEX idx_audit_logs_v2_status ON audit_logs_v2(status);
CREATE INDEX idx_audit_logs_v2_created_at ON audit_logs_v2(created_at DESC);
CREATE INDEX idx_audit_logs_v2_request_id ON audit_logs_v2(request_id) WHERE request_id IS NOT NULL;

-- Add constraints
ALTER TABLE audit_logs_v2 ADD CONSTRAINT check_audit_v2_actor_type
    CHECK (actor_type IN ('system', 'merchant', 'admin', 'ops', 'api', 'worker', 'listener'));

ALTER TABLE audit_logs_v2 ADD CONSTRAINT check_audit_v2_action_category
    CHECK (action_category IN ('payment', 'payout', 'kyc', 'authentication', 'admin', 'system', 'blockchain', 'webhook', 'security'));

ALTER TABLE audit_logs_v2 ADD CONSTRAINT check_audit_v2_status
    CHECK (status IN ('success', 'failed', 'error'));

COMMENT ON TABLE audit_logs_v2 IS 'Partitioned audit logs (v2) - will replace audit_logs after migration';
```

#### Step 1.2: Test Migration on Staging
```bash
# Create staging database with production-like data
pg_dump production_db -t audit_logs > audit_logs_backup.sql
psql staging_db < audit_logs_backup.sql

# Run migration
psql staging_db -f migrations/013_create_audit_logs_partitioned.up.sql

# Verify partitions created
psql staging_db -c "SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE tablename LIKE 'audit_logs%'
ORDER BY tablename;"

# Expected output:
# audit_logs          | 500 MB
# audit_logs_v2       | 0 bytes (empty)
# audit_logs_v2_2025  | 0 bytes
# audit_logs_v2_2026  | 0 bytes
# ...
```

---

### **Phase 2: Dual-Write Implementation** (Week 0, Day 1-2)

#### Step 2.1: Update Audit Repository

```go
// internal/repository/audit.go

type AuditRepository interface {
    Create(log *model.AuditLog) error
    // ... other methods
}

type auditRepository struct {
    db              *sql.DB
    dualWriteMode   bool  // NEW: Enable dual-write during migration
}

func (r *auditRepository) Create(log *model.AuditLog) error {
    // Write to old table (audit_logs)
    err := r.writeToOldTable(log)
    if err != nil {
        return fmt.Errorf("failed to write to audit_logs: %w", err)
    }

    // NEW: Also write to new partitioned table (if dual-write enabled)
    if r.dualWriteMode {
        err = r.writeToNewTable(log)
        if err != nil {
            // Log error but DON'T fail the request
            // We can backfill this later
            logrus.WithError(err).Warn("Failed to dual-write to audit_logs_v2 (will backfill)")
        }
    }

    return nil
}

func (r *auditRepository) writeToOldTable(log *model.AuditLog) error {
    query := `
        INSERT INTO audit_logs (
            id, actor_type, actor_id, action, action_category,
            resource_type, resource_id, status, metadata, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `
    _, err := r.db.Exec(query,
        log.ID, log.ActorType, log.ActorID, log.Action, log.ActionCategory,
        log.ResourceType, log.ResourceID, log.Status, log.Metadata, log.CreatedAt,
    )
    return err
}

func (r *auditRepository) writeToNewTable(log *model.AuditLog) error {
    query := `
        INSERT INTO audit_logs_v2 (
            id, actor_type, actor_id, action, action_category,
            resource_type, resource_id, status, metadata, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `
    _, err := r.db.Exec(query,
        log.ID, log.ActorType, log.ActorID, log.Action, log.ActionCategory,
        log.ResourceType, log.ResourceID, log.Status, log.Metadata, log.CreatedAt,
    )
    return err
}
```

#### Step 2.2: Enable Dual-Write via Config
```bash
# .env
AUDIT_DUAL_WRITE_MODE=true  # Enable dual-write to audit_logs_v2
```

#### Step 2.3: Deploy Dual-Write Code
```bash
# Deploy new code with dual-write enabled
git pull
make build
systemctl restart payment-gateway-api
systemctl restart payment-gateway-listener
systemctl restart payment-gateway-worker

# Verify dual-write working
tail -f /var/log/payment-gateway/app.log | grep "dual-write"
```

---

### **Phase 3: Backfill Historical Data** (Week 0, Day 3-4)

#### Step 3.1: Create Backfill Script
```sql
-- scripts/backfill_audit_logs.sql

-- Copy data in batches (to avoid locking)
DO $$
DECLARE
    batch_size INT := 10000;
    total_rows BIGINT;
    processed BIGINT := 0;
    start_time TIMESTAMP;
BEGIN
    -- Get total rows to migrate
    SELECT COUNT(*) INTO total_rows FROM audit_logs;
    RAISE NOTICE 'Total rows to migrate: %', total_rows;

    start_time := clock_timestamp();

    -- Loop through batches
    LOOP
        -- Insert batch
        INSERT INTO audit_logs_v2
        SELECT * FROM audit_logs
        WHERE id NOT IN (SELECT id FROM audit_logs_v2)
        ORDER BY created_at ASC
        LIMIT batch_size;

        -- Check if any rows were inserted
        IF NOT FOUND THEN
            EXIT;
        END IF;

        processed := processed + batch_size;
        RAISE NOTICE 'Processed % / % rows (%.2f%%) - Elapsed: %',
            processed, total_rows,
            (processed::FLOAT / total_rows * 100),
            clock_timestamp() - start_time;

        -- Small delay to avoid overwhelming database
        PERFORM pg_sleep(0.1);
    END LOOP;

    RAISE NOTICE 'Backfill complete! Total time: %', clock_timestamp() - start_time;
END $$;
```

#### Step 3.2: Run Backfill (Off-Peak Hours)
```bash
# Run at 2am when traffic is low
psql payment_gateway -f scripts/backfill_audit_logs.sql

# Monitor progress
psql payment_gateway -c "
SELECT
    'audit_logs' as table_name,
    COUNT(*) as row_count
FROM audit_logs
UNION ALL
SELECT
    'audit_logs_v2' as table_name,
    COUNT(*) as row_count
FROM audit_logs_v2;
"

# Expected output (during backfill):
# table_name        | row_count
# ------------------+-----------
# audit_logs        | 1,234,567
# audit_logs_v2     | 823,456   (growing)
```

---

### **Phase 4: Verification** (Week 0, Day 5)

#### Step 4.1: Verify Data Integrity
```sql
-- Check row counts match
SELECT
    (SELECT COUNT(*) FROM audit_logs) as old_count,
    (SELECT COUNT(*) FROM audit_logs_v2) as new_count,
    (SELECT COUNT(*) FROM audit_logs) - (SELECT COUNT(*) FROM audit_logs_v2) as difference;

-- Expected: difference = 0

-- Check sample data matches
SELECT
    a1.id,
    a1.action,
    a1.created_at,
    a2.action as v2_action,
    a2.created_at as v2_created_at,
    CASE
        WHEN a1.action = a2.action AND a1.created_at = a2.created_at THEN 'MATCH'
        ELSE 'MISMATCH'
    END as status
FROM audit_logs a1
JOIN audit_logs_v2 a2 ON a1.id = a2.id
ORDER BY RANDOM()
LIMIT 100;

-- Expected: All rows show 'MATCH'

-- Check partition distribution
SELECT
    tableoid::regclass AS partition_name,
    COUNT(*) as row_count,
    MIN(created_at) as oldest_record,
    MAX(created_at) as newest_record
FROM audit_logs_v2
GROUP BY tableoid
ORDER BY partition_name;

-- Expected output:
# partition_name         | row_count | oldest_record | newest_record
# -----------------------+-----------+---------------+---------------
# audit_logs_v2_2025     | 1,234,567 | 2025-01-01    | 2025-11-18
# audit_logs_v2_2026     | 0         | NULL          | NULL
```

#### Step 4.2: Performance Test
```sql
-- Test query performance on old table
EXPLAIN ANALYZE
SELECT * FROM audit_logs
WHERE created_at > '2025-11-01'
  AND action_category = 'payment'
ORDER BY created_at DESC
LIMIT 100;

-- Test query performance on new partitioned table
EXPLAIN ANALYZE
SELECT * FROM audit_logs_v2
WHERE created_at > '2025-11-01'
  AND action_category = 'payment'
ORDER BY created_at DESC
LIMIT 100;

-- Expected: audit_logs_v2 should be FASTER (partition pruning)
```

---

### **Phase 5: Switch Reads** (Week 1, Day 1)

#### Step 5.1: Update Repository to Read from New Table
```go
// internal/repository/audit.go

type auditRepository struct {
    db              *sql.DB
    dualWriteMode   bool
    readFromV2      bool  // NEW: Switch reads to v2
}

func (r *auditRepository) GetByID(id string) (*model.AuditLog, error) {
    tableName := "audit_logs"
    if r.readFromV2 {
        tableName = "audit_logs_v2"
    }

    query := fmt.Sprintf(`
        SELECT id, actor_type, action, resource_type, resource_id, created_at
        FROM %s
        WHERE id = $1
    `, tableName)

    var log model.AuditLog
    err := r.db.QueryRow(query, id).Scan(&log.ID, &log.ActorType, ...)
    return &log, err
}
```

#### Step 5.2: Enable Read from V2
```bash
# .env
AUDIT_DUAL_WRITE_MODE=true
AUDIT_READ_FROM_V2=true     # NEW: Switch reads to partitioned table

# Deploy
make build && systemctl restart payment-gateway-*

# Monitor for errors
tail -f /var/log/payment-gateway/app.log | grep -i "audit"
```

---

### **Phase 6: Final Cutover** (Week 1, Day 2-3)

#### Step 6.1: Stop Dual-Write (Write Only to V2)
```go
// Remove old table writes
func (r *auditRepository) Create(log *model.AuditLog) error {
    // Write ONLY to new table
    return r.writeToNewTable(log)
}
```

#### Step 6.2: Rename Tables (Atomic Swap)
```sql
-- migrations/013_rename_audit_logs_v2.up.sql

BEGIN;
    -- Rename old table to _archived
    ALTER TABLE audit_logs RENAME TO audit_logs_archived;

    -- Rename new table to audit_logs
    ALTER TABLE audit_logs_v2 RENAME TO audit_logs;

    -- Rename partitions (PostgreSQL auto-updates partition names)
    ALTER TABLE audit_logs_2025 RENAME TO audit_logs_2025;

    -- Update code to use "audit_logs" (no v2 suffix)
COMMIT;
```

#### Step 6.3: Verify Production
```bash
# Check table exists
psql payment_gateway -c "\dt audit_logs*"

# Expected:
# audit_logs            (partitioned table)
# audit_logs_2025       (partition)
# audit_logs_2026       (partition)
# audit_logs_archived   (old table, will be dropped)

# Verify app still works
curl http://localhost:8080/health
```

---

### **Phase 7: Cleanup** (Week 2)

#### Step 7.1: Keep Old Table for 1 Week (Safety)
```bash
# Monitor for any issues for 1 week
# If everything works fine, drop old table

# After 1 week of confidence:
psql payment_gateway -c "DROP TABLE audit_logs_archived;"
```

---

## 🔄 Rollback Procedures

### **Rollback from Phase 2 (Dual-Write)**
```bash
# Disable dual-write
AUDIT_DUAL_WRITE_MODE=false
systemctl restart payment-gateway-*

# Drop new table (data not yet critical)
psql payment_gateway -c "DROP TABLE audit_logs_v2 CASCADE;"
```

### **Rollback from Phase 5 (Reads Switched)**
```bash
# Switch reads back to old table
AUDIT_READ_FROM_V2=false
systemctl restart payment-gateway-*

# Continue dual-write until issue resolved
```

### **Rollback from Phase 6 (After Cutover)**
```bash
# Emergency rollback (if critical issues found)
psql payment_gateway <<EOF
BEGIN;
    ALTER TABLE audit_logs RENAME TO audit_logs_v2_temp;
    ALTER TABLE audit_logs_archived RENAME TO audit_logs;
    -- App now reads from old table
COMMIT;
EOF

# Update code to read from old table
AUDIT_READ_FROM_V2=false
systemctl restart payment-gateway-*
```

---

## ✅ Success Criteria

- [ ] Zero data loss (row counts match)
- [ ] Zero downtime (application always available)
- [ ] Query performance improved (partition pruning working)
- [ ] Rollback tested on staging
- [ ] Old table archived for 1 week before deletion

---

## 📊 Timeline Summary

| Phase | Duration | Risk | Rollback |
|-------|----------|------|----------|
| 1. Preparation | 1 week | Low | Easy |
| 2. Dual-Write | 2 days | Low | Easy |
| 3. Backfill | 2 days | Medium | Easy |
| 4. Verification | 1 day | Low | Easy |
| 5. Switch Reads | 1 day | Medium | Medium |
| 6. Final Cutover | 1 day | High | Hard |
| 7. Cleanup | 1 week | Low | N/A |

**Total: ~2 weeks** (with 1 week safety buffer)

---

This approach is **production-safe** and used by companies like GitHub, Stripe, and Shopify for critical table migrations.
