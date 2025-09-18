-- +migrate Up
CREATE TABLE IF NOT EXISTS roles(
                                    id BIGINT AUTO_INCREMENT PRIMARY KEY,
                                    name VARCHAR(50) NOT NULL UNIQUE
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS user_roles(
                                         user_id BIGINT NOT NULL,
                                         role_id BIGINT NOT NULL,
                                         PRIMARY KEY(user_id, role_id)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO roles(name) VALUES ('admin'),('user');
-- +migrate Down
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;
