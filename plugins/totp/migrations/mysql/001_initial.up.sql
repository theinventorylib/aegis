-- TOTP plugin schema for MySQL

ALTER TABLE `user` ADD COLUMN totp_secret TEXT;
ALTER TABLE `user` ADD COLUMN totp_enabled TINYINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS totp_session (
    session_id  VARCHAR(255) PRIMARY KEY,
    user_id     VARCHAR(255) NOT NULL,
    verified_at VARCHAR(255) NOT NULL,
    FOREIGN KEY (session_id) REFERENCES session(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES `user`(id) ON DELETE CASCADE
);

CREATE INDEX idx_totp_session_user ON totp_session(user_id);
