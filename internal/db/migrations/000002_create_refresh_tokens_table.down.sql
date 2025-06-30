-- Drop trigger
DROP TRIGGER IF EXISTS update_refresh_tokens_last_used_at ON refresh_tokens;

-- Drop indexes
DROP INDEX IF EXISTS idx_refresh_tokens_user_token;
DROP INDEX IF EXISTS idx_refresh_tokens_expires_at;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;

-- Drop table
DROP TABLE IF EXISTS refresh_tokens;