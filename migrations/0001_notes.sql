-- +migrate Up
CREATE TABLE IF NOT EXISTS notes(
                                    id BIGINT AUTO_INCREMENT PRIMARY KEY,
                                    user_id BIGINT NOT NULL,
                                    title TEXT NOT NULL,
                                    body  TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +migrate Down
DROP TABLE IF EXISTS notes;