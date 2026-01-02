-- ==============================================
-- Core Table 1: user
-- ==============================================
CREATE TABLE IF NOT EXISTS "user" (
    id TEXT PRIMARY KEY,
    avatar TEXT,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    disabled INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_user_disabled ON "user"(disabled);
CREATE INDEX IF NOT EXISTS idx_user_email ON "user"(email);

