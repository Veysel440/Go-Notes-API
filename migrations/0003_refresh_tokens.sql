-- +migrate Up
CREATE TABLE IF NOT EXISTS refresh_tokens(
                                             token VARCHAR(64) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    expires_at DATETIME NOT NULL,
    INDEX ix_refresh_expires (expires_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +migrate Down
DROP TABLE IF EXISTS refresh_tokens;