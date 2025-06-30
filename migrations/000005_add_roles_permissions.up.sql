-- Create roles and permissions tables
BEGIN;

-- Create roles table
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create permissions table
CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create role_permissions junction table
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_id)
);

-- Create user_roles junction table
CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    assigned_by UUID REFERENCES users(id),
    PRIMARY KEY (user_id, role_id)
);

-- Add indexes
CREATE INDEX idx_permissions_resource_action ON permissions(resource, action);
CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission_id ON role_permissions(permission_id);

-- Insert default roles
INSERT INTO roles (name, description, is_system) VALUES
    ('admin', 'System administrator with full access', true),
    ('user', 'Regular user with basic access', true),
    ('moderator', 'Moderator with content management access', true);

-- Insert default permissions
INSERT INTO permissions (name, resource, action, description) VALUES
    ('users.read', 'users', 'read', 'View user profiles'),
    ('users.write', 'users', 'write', 'Edit user profiles'),
    ('users.delete', 'users', 'delete', 'Delete user accounts'),
    ('users.list', 'users', 'list', 'List all users'),
    ('roles.read', 'roles', 'read', 'View roles'),
    ('roles.write', 'roles', 'write', 'Create and edit roles'),
    ('roles.delete', 'roles', 'delete', 'Delete roles'),
    ('roles.assign', 'roles', 'assign', 'Assign roles to users'),
    ('audit.read', 'audit', 'read', 'View audit logs'),
    ('system.admin', 'system', 'admin', 'Full system access');

-- Assign permissions to roles
-- Admin gets all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin';

-- User gets basic read permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'user' AND p.name IN ('users.read');

-- Moderator gets user management permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'moderator' AND p.name IN ('users.read', 'users.list', 'audit.read');

-- Add trigger to update roles.updated_at
CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMIT;