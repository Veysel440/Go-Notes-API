-- +migrate Up
ALTER TABLE idempotency_keys ADD COLUMN IF NOT EXISTS result_text TEXT NULL;
-- +migrate Down
ALTER TABLE idempotency_keys DROP COLUMN IF EXISTS result_text;