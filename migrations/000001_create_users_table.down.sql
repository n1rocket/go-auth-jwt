-- Drop trigger
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_users_password_reset_token;
DROP INDEX IF EXISTS idx_users_email_verification_token;
DROP INDEX IF EXISTS idx_users_email;

-- Drop table
DROP TABLE IF EXISTS users;