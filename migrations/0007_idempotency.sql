-- +migrate Up
CREATE TABLE IF NOT EXISTS idempotency_keys(
                                               `key`       VARCHAR(80) PRIMARY KEY,
    user_id     BIGINT NOT NULL,
    method      VARCHAR(8) NOT NULL,
    path        TEXT NOT NULL,
    body_sha256 CHAR(64) NOT NULL,
    note_id     BIGINT NULL,
    claimed_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME NULL
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX IF NOT EXISTS ix_idem_user ON idempotency_keys(user_id);

-- +migrate Down
DROP TABLE IF EXISTS idempotency_keys;
