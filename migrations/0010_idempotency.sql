-- +migrate Up
CREATE TABLE IF NOT EXISTS idempotency_keys (
                                                id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
                                                `key`        VARCHAR(128)    NOT NULL,
    user_id      BIGINT          NOT NULL DEFAULT 0,
    method       VARCHAR(10)     NOT NULL,
    path         VARCHAR(255)    NOT NULL,
    body_sha256  CHAR(64)        NOT NULL,
    result_text  MEDIUMTEXT      NULL,
    claimed_at   DATETIME(6)     NOT NULL,
    completed_at DATETIME(6)     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY ux_idem_user_key (user_id, `key`),
    KEY          ix_idem_user_completed (user_id, completed_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +migrate Down
DROP TABLE IF EXISTS idempotency_keys;
