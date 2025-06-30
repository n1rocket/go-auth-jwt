-- Remove roles and permissions tables
BEGIN;

-- Drop trigger
DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;

-- Drop indexes
DROP INDEX IF EXISTS idx_permissions_resource_action;
DROP INDEX IF EXISTS idx_user_roles_user_id;
DROP INDEX IF EXISTS idx_user_roles_role_id;
DROP INDEX IF EXISTS idx_role_permissions_role_id;
DROP INDEX IF EXISTS idx_role_permissions_permission_id;

-- Drop junction tables first (due to foreign key constraints)
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;

-- Drop main tables
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;

COMMIT;