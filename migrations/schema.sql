-- ==============================================
-- Aegis Core Schema v2.0
-- 5 Core Tables: user, accounts, verification, session, jwks
-- Everything else is plugin-managed
-- ==============================================

-- Drop old tables (will be migrated)
DROP TABLE IF EXISTS auth.password_reset_tokens CASCADE;
DROP TABLE IF EXISTS auth.otps CASCADE;
DROP TABLE IF EXISTS auth.oauth_providers CASCADE;

-- Rename existing tables for migration
ALTER TABLE IF EXISTS auth.users RENAME TO users_old;
ALTER TABLE IF EXISTS auth.sessions RENAME TO sessions_old;

-- ==============================================
-- Core Table 1: user
-- ==============================================
CREATE TABLE auth.user (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_user_metadata ON auth.user USING GIN(metadata);

-- ==============================================
-- Core Table 2: accounts
-- ==============================================
CREATE TABLE auth.accounts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES auth.user(id) ON DELETE CASCADE,
    provider TEXT NOT NULL, -- "password", "google", "github", "apple", etc.
    provider_account_id TEXT, -- NULL for password, OAuth user ID for providers
    password_hash TEXT, -- Only populated for "password" provider
    access_token TEXT,
    refresh_token TEXT,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}',
    UNIQUE(provider, provider_account_id)
);

CREATE INDEX idx_accounts_user_id ON auth.accounts(user_id);
CREATE INDEX idx_accounts_provider ON auth.accounts(provider);
CREATE INDEX idx_accounts_metadata ON auth.accounts USING GIN(metadata);

-- ==============================================
-- Core Table 3: verification
-- ==============================================
CREATE TABLE auth.verification (
    id TEXT PRIMARY KEY,
    identifier TEXT NOT NULL, -- email, phone, user_id depending on type
    token TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL, -- "email_verification", "password_reset", "magic_link"
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_verification_identifier ON auth.verification(identifier);
CREATE INDEX idx_verification_token ON auth.verification(token);
CREATE INDEX idx_verification_type ON auth.verification(type);
CREATE INDEX idx_verification_expires_at ON auth.verification(expires_at);

-- ==============================================
-- Core Table 4: session
-- ==============================================
CREATE TABLE auth.session (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES auth.user(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    refresh_token TEXT UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ip_address TEXT,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_session_user_id ON auth.session(user_id);
CREATE INDEX idx_session_token ON auth.session(token);
CREATE INDEX idx_session_refresh_token ON auth.session(refresh_token);
CREATE INDEX idx_session_expires_at ON auth.session(expires_at);
CREATE INDEX idx_session_metadata ON auth.session USING GIN(metadata);

-- ==============================================
-- Core Table 5: jwks
-- ==============================================
CREATE TABLE auth.jwks (
    kid TEXT PRIMARY KEY,
    key_data JSONB NOT NULL,
    algorithm TEXT NOT NULL,
    use TEXT DEFAULT 'sig', -- 'sig' for signing, 'enc' for encryption
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP
);

CREATE INDEX idx_jwks_algorithm ON auth.jwks(algorithm);
CREATE INDEX idx_jwks_use ON auth.jwks(use);

-- ==============================================
-- Triggers for updated_at
-- ==============================================
CREATE OR REPLACE FUNCTION auth.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_user_updated_at
    BEFORE UPDATE ON auth.user
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();

CREATE TRIGGER update_accounts_updated_at
    BEFORE UPDATE ON auth.accounts
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();

-- ==============================================
-- Data Migration from Old Schema
-- ==============================================

-- Migrate users
INSERT INTO auth.user (id, email, email_verified, created_at, updated_at)
SELECT id, email, 
       COALESCE(email_verified, FALSE), 
       created_at, 
       COALESCE(updated_at, created_at)
FROM users_old
WHERE EXISTS (SELECT 1 FROM users_old);

-- Migrate password accounts
INSERT INTO auth.accounts (user_id, provider, password_hash, created_at, updated_at)
SELECT id, 'password', password_hash, created_at, COALESCE(updated_at, created_at)
FROM users_old
WHERE password_hash IS NOT NULL
  AND EXISTS (SELECT 1 FROM users_old WHERE password_hash IS NOT NULL);

-- Migrate sessions
INSERT INTO auth.session (id, user_id, token, refresh_token, expires_at, created_at, ip_address, user_agent)
SELECT id, user_id, token, refresh_token, expires_at, created_at, 
       COALESCE(ip_address, ''), 
       COALESCE(user_agent, '')
FROM sessions_old
WHERE EXISTS (SELECT 1 FROM sessions_old);

-- ==============================================
-- Cleanup (optional - comment out for safety)
-- ==============================================
-- DROP TABLE IF EXISTS users_old CASCADE;
-- DROP TABLE IF EXISTS sessions_old CASCADE;
