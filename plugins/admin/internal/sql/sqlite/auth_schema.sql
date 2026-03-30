-- ==============================================
-- Aegis Core Schema v3.0 (SQLite)
-- 4 Core Tables: user, accounts, verification, session
-- Created specifically for SQLite
-- Used with sqlc by default
-- ==============================================

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

-- ==============================================
-- Core Table 2: accounts
-- ==============================================
CREATE TABLE IF NOT EXISTS accounts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_account_id TEXT,
    password_hash TEXT,
    access_token TEXT,
    refresh_token TEXT,
    expires_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(provider, provider_account_id),
    FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_accounts_provider ON accounts(provider);

-- ==============================================
-- Core Table 3: verification
-- ==============================================
CREATE TABLE IF NOT EXISTS verification (
    id TEXT PRIMARY KEY,
    identifier TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_verification_identifier ON verification(identifier);
CREATE INDEX IF NOT EXISTS idx_verification_token ON verification(token);
CREATE INDEX IF NOT EXISTS idx_verification_type ON verification(type);
CREATE INDEX IF NOT EXISTS idx_verification_expires_at ON verification(expires_at);

-- ==============================================
-- Core Table 4: session
-- ==============================================
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

CREATE INDEX IF NOT EXISTS idx_session_user_id ON session(user_id);
CREATE INDEX IF NOT EXISTS idx_session_token ON session(token);
CREATE INDEX IF NOT EXISTS idx_session_refresh_token ON session(refresh_token);
CREATE INDEX IF NOT EXISTS idx_session_expires_at ON session(expires_at);