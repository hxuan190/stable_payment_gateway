-- Drop indexes
DROP INDEX IF EXISTS idx_audit_logs_resource_action;
DROP INDEX IF EXISTS idx_audit_logs_actor_action;
DROP INDEX IF EXISTS idx_audit_logs_actor_ip;
DROP INDEX IF EXISTS idx_audit_logs_request_id;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_status;
DROP INDEX IF EXISTS idx_audit_logs_resource;
DROP INDEX IF EXISTS idx_audit_logs_action_category;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_actor_type;
DROP INDEX IF EXISTS idx_audit_logs_actor_id;

-- Drop table
DROP TABLE IF EXISTS audit_logs;
