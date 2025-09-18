-- +migrate Up
ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS used_at DATETIME NULL;
-- +migrate Down
ALTER TABLE refresh_tokens DROP COLUMN used_at;
