-- Remove audit tables
BEGIN;

-- Drop trigger
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_audit_logs_user_id;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_resource;

DROP INDEX IF EXISTS idx_login_attempts_email;
DROP INDEX IF EXISTS idx_login_attempts_ip;
DROP INDEX IF EXISTS idx_login_attempts_created_at;

DROP INDEX IF EXISTS idx_password_reset_tokens_user_id;
DROP INDEX IF EXISTS idx_password_reset_tokens_token;
DROP INDEX IF EXISTS idx_password_reset_tokens_expires_at;

-- Drop tables
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS login_attempts;
DROP TABLE IF EXISTS audit_logs;

COMMIT;