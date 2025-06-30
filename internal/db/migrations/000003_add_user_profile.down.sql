-- Remove user profile fields
BEGIN;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_last_login_at;
DROP INDEX IF EXISTS idx_users_created_at;

-- Remove check constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_login_count_positive;

-- Remove columns
ALTER TABLE users 
DROP COLUMN IF EXISTS first_name,
DROP COLUMN IF EXISTS last_name,
DROP COLUMN IF EXISTS phone_number,
DROP COLUMN IF EXISTS avatar_url,
DROP COLUMN IF EXISTS bio,
DROP COLUMN IF EXISTS date_of_birth,
DROP COLUMN IF EXISTS last_login_at,
DROP COLUMN IF EXISTS login_count,
DROP COLUMN IF EXISTS is_active,
DROP COLUMN IF EXISTS metadata;

COMMIT;