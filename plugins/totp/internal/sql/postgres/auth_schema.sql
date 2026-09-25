-- ==============================================
-- Core auth schema snapshot for sqlc (PostgreSQL)
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


CREATE TABLE IF NOT EXISTS session (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    refresh_token TEXT UNIQUE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);
