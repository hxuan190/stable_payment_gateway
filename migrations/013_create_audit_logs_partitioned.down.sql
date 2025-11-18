-- Rollback partitioned audit logs table
-- This is safe to run if migration was just applied (no data loss from original audit_logs)

-- Drop composite indexes
DROP INDEX IF EXISTS idx_audit_logs_v2_resource_action;
DROP INDEX IF EXISTS idx_audit_logs_v2_actor_action;

-- Drop indexes
DROP INDEX IF EXISTS idx_audit_logs_v2_actor_ip;
DROP INDEX IF EXISTS idx_audit_logs_v2_request_id;
DROP INDEX IF EXISTS idx_audit_logs_v2_created_at;
DROP INDEX IF EXISTS idx_audit_logs_v2_status;
DROP INDEX IF EXISTS idx_audit_logs_v2_resource;
DROP INDEX IF EXISTS idx_audit_logs_v2_action_category;
DROP INDEX IF EXISTS idx_audit_logs_v2_action;
DROP INDEX IF EXISTS idx_audit_logs_v2_actor_type;
DROP INDEX IF EXISTS idx_audit_logs_v2_actor_id;

-- Drop partitions (CASCADE will drop them when parent table is dropped)
-- But explicitly dropping for clarity
DROP TABLE IF EXISTS audit_logs_v2_2030;
DROP TABLE IF EXISTS audit_logs_v2_2029;
DROP TABLE IF EXISTS audit_logs_v2_2028;
DROP TABLE IF EXISTS audit_logs_v2_2027;
DROP TABLE IF EXISTS audit_logs_v2_2026;
DROP TABLE IF EXISTS audit_logs_v2_2025;

-- Drop partitioned table (CASCADE will drop all partitions)
DROP TABLE IF EXISTS audit_logs_v2 CASCADE;

-- Note: Original audit_logs table remains untouched
-- This rollback is safe as long as dual-write hasn't been enabled yet
