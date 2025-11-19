-- Rollback: Drop notification tables

-- Drop views
DROP VIEW IF EXISTS pending_retry_notifications CASCADE;
DROP VIEW IF EXISTS notification_delivery_stats CASCADE;

-- Drop triggers and functions
DROP TRIGGER IF EXISTS trigger_update_merchant_prefs_updated_at ON merchant_notification_preferences;
DROP TRIGGER IF EXISTS trigger_update_notification_logs_updated_at ON notification_logs;
DROP FUNCTION IF EXISTS update_merchant_prefs_updated_at();
DROP FUNCTION IF EXISTS update_notification_logs_updated_at();

-- Drop tables
DROP TABLE IF EXISTS merchant_notification_preferences CASCADE;
DROP TABLE IF EXISTS notification_logs CASCADE;
