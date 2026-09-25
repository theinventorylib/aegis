-- TOTP plugin schema for SQLite

ALTER TABLE "user" ADD COLUMN totp_secret TEXT;
ALTER TABLE "user" ADD COLUMN totp_enabled INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS totp_session (
    session_id  TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    verified_at TEXT NOT NULL,
    FOREIGN KEY (session_id) REFERENCES session(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_totp_session_user ON totp_session(user_id);
