-- +migrate Up
CREATE TABLE IF NOT EXISTS audit_logs(
                                         id BIGINT AUTO_INCREMENT PRIMARY KEY,
                                         user_id BIGINT NULL,
                                         method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    status INT NOT NULL,
    ip VARCHAR(64) NOT NULL,
    rid VARCHAR(64) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +migrate Down
DROP TABLE IF EXISTS audit_logs;
