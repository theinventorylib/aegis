-- ==============================================
-- Core Table 1: user
-- ==============================================
CREATE TABLE IF NOT EXISTS `user` (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE,
    avatar VARCHAR(255),
    disabled TINYINT NOT NULL DEFAULT 0,
    created_at VARCHAR(255) NOT NULL,
    updated_at VARCHAR(255) NOT NULL
);

CREATE INDEX idx_user_disabled ON `user`(disabled);
CREATE INDEX idx_user_email ON `user`(email);