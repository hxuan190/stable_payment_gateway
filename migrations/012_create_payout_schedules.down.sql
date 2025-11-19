-- Rollback: Drop payout_schedules table

-- Drop views
DROP VIEW IF EXISTS payout_schedule_summary CASCADE;
DROP VIEW IF EXISTS threshold_payout_configs CASCADE;
DROP VIEW IF EXISTS upcoming_scheduled_payouts CASCADE;

-- Drop functions
DROP FUNCTION IF EXISTS get_next_scheduled_payout(UUID);

-- Drop triggers and functions
DROP TRIGGER IF EXISTS trigger_update_payout_schedules_updated_at ON payout_schedules;
DROP FUNCTION IF EXISTS update_payout_schedules_updated_at();

-- Drop table
DROP TABLE IF EXISTS payout_schedules CASCADE;
