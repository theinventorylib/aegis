-- TOTP plugin schema for PostgreSQL

-- Add TOTP fields to the user table. The secret is written at setup and only
-- trusted after a code is confirmed (totp_enabled = 1).
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS totp_secret TEXT;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS totp_enabled INTEGER NOT NULL DEFAULT 0;

-- Per-session second-factor state. Rows cascade with the session, so logout
-- or session expiry can never leave a verified flag behind.
CREATE TABLE IF NOT EXISTS totp_session (
    session_id  TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    verified_at TEXT NOT NULL,
    FOREIGN KEY (session_id) REFERENCES session(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_totp_session_user ON totp_session(user_id);
