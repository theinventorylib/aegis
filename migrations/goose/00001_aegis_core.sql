-- +goose Up
-- Create auth schema
CREATE SCHEMA IF NOT EXISTS auth;

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION auth.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ==============================================
-- Core Table 1: user
-- ==============================================
CREATE TABLE auth.user (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    email TEXT UNIQUE NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_user_email ON auth.user(email);
CREATE INDEX idx_user_metadata ON auth.user USING GIN(metadata);

CREATE TRIGGER update_user_updated_at
    BEFORE UPDATE ON auth.user
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();

-- ==============================================
-- Core Table 2: accounts  
-- ==============================================
CREATE TABLE auth.accounts (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id TEXT NOT NULL REFERENCES auth.user(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_account_id TEXT,
    password_hash TEXT,
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

CREATE TRIGGER update_accounts_updated_at
    BEFORE UPDATE ON auth.accounts
    FOR EACH ROW
    EXECUTE FUNCTION auth.update_updated_at_column();

-- ==============================================
-- Core Table 3: verification
-- ==============================================
CREATE TABLE auth.verification (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    identifier TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
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
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
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

-- +goose Down
DROP TABLE IF EXISTS auth.session;
DROP TABLE IF EXISTS auth.verification;
DROP TABLE IF EXISTS auth.accounts;
DROP TABLE IF EXISTS auth.user;
DROP FUNCTION IF EXISTS auth.update_updated_at_column();
DROP SCHEMA IF EXISTS auth CASCADE;
