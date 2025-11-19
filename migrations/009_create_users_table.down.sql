-- Rollback: Drop users table

-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_users_updated_at();

-- Drop table
DROP TABLE IF EXISTS users CASCADE;
