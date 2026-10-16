-- ==============================================
-- Aegis Core Schema v3.0
-- 4 Core Tables: user, accounts, verification, session
-- Created specifically for MySQL
-- User with sqlc by default
--
-- Columns used in keys, uniques or foreign keys are VARCHAR: MySQL requires
-- a bounded width for indexed columns (TEXT can only be indexed with a prefix).
-- ==============================================

-- ==============================================
-- Core Table 1: user
-- ==============================================
CREATE TABLE IF NOT EXISTS `user` (
    id VARCHAR(255) PRIMARY KEY,
    avatar TEXT,
    name TEXT NOT NULL,
    email VARCHAR(255) UNIQUE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    disabled TINYINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_user_disabled ON `user`(disabled);
CREATE INDEX idx_user_email ON `user`(email);

-- ==============================================
-- Core Table 2: accounts
-- ==============================================
CREATE TABLE IF NOT EXISTS accounts (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    provider VARCHAR(255) NOT NULL,
    provider_account_id VARCHAR(255),
    password_hash TEXT,
    access_token TEXT,
    refresh_token TEXT,
    expires_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(provider, provider_account_id),
    FOREIGN KEY (user_id) REFERENCES `user`(id) ON DELETE CASCADE
);

CREATE INDEX idx_accounts_user_id ON accounts(user_id);
CREATE INDEX idx_accounts_provider ON accounts(provider);

-- ==============================================
-- Core Table 3: verification
-- ==============================================
CREATE TABLE IF NOT EXISTS verification (
    id VARCHAR(255) PRIMARY KEY,
    identifier VARCHAR(255) NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(255) NOT NULL,
    expires_at VARCHAR(255) NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX idx_verification_identifier ON verification(identifier);
CREATE INDEX idx_verification_token ON verification(token);
CREATE INDEX idx_verification_type ON verification(type);
CREATE INDEX idx_verification_expires_at ON verification(expires_at);

-- ==============================================
-- Core Table 4: session
-- ==============================================
CREATE TABLE IF NOT EXISTS session (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    refresh_token VARCHAR(255) UNIQUE,
    expires_at VARCHAR(255) NOT NULL,
    created_at TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    FOREIGN KEY (user_id) REFERENCES `user`(id) ON DELETE CASCADE
);

CREATE INDEX idx_session_user_id ON session(user_id);
CREATE INDEX idx_session_token ON session(token);
CREATE INDEX idx_session_refresh_token ON session(refresh_token);
CREATE INDEX idx_session_expires_at ON session(expires_at);
