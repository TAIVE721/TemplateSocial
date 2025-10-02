CREATE TABLE IF NOT EXISTS roles (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    level INT NOT NULL DEFAULT 0
);

-- Insertamos los roles por defecto
INSERT INTO roles (name, level) VALUES ('user', 1);

INSERT INTO roles (name, level) VALUES ('moderator', 2);

INSERT INTO roles (name, level) VALUES ('admin', 3);