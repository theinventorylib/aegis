-- JWT plugin schema for SQLite
-- JWKS table for storing JSON Web Keys
CREATE TABLE IF NOT EXISTS jwks (
    kid TEXT PRIMARY KEY,
    key_data TEXT NOT NULL,
    algorithm TEXT NOT NULL,
    use TEXT DEFAULT 'sig',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at TEXT
);

-- Indexes for efficient key lookup
CREATE INDEX IF NOT EXISTS idx_jwks_algorithm ON jwks(algorithm);
CREATE INDEX IF NOT EXISTS idx_jwks_use ON jwks(use);
CREATE INDEX IF NOT EXISTS idx_jwks_expires_at ON jwks(expires_at);
CREATE INDEX IF NOT EXISTS idx_jwks_created_at ON jwks(created_at);