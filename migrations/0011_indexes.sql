-- +migrate Up
-- audit_logs: id, user_id, action, ip, ua, created_at, meta_json
CREATE INDEX IF NOT EXISTS ix_audit_user_time ON audit_logs (user_id, created_at);
CREATE INDEX IF NOT EXISTS ix_audit_action_time ON audit_logs (action, created_at);

-- user_roles: id, user_id, role, created_at
CREATE UNIQUE INDEX IF NOT EXISTS ux_user_roles_user_role ON user_roles (user_id, role);
CREATE INDEX        IF NOT EXISTS ix_user_roles_role      ON user_roles (role);

-- +migrate Down
DROP INDEX IF EXISTS ix_audit_user_time ON audit_logs;
DROP INDEX IF EXISTS ix_audit_action_time ON audit_logs;
DROP INDEX IF EXISTS ux_user_roles_user_role ON user_roles;
DROP INDEX IF EXISTS ix_user_roles_role ON user_roles;