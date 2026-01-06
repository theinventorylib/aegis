-- JWT plugin schema for MySQL
-- JWKS table for storing JSON Web Keys
CREATE TABLE IF NOT EXISTS jwks (
    kid VARCHAR(255) PRIMARY KEY,
    key_data LONGTEXT NOT NULL,
    algorithm VARCHAR(50) NOT NULL,
    use VARCHAR(10) DEFAULT 'sig',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NULL
);

-- Indexes for efficient key lookup
CREATE INDEX idx_jwks_algorithm ON jwks(algorithm);
CREATE INDEX idx_jwks_use ON jwks(use);
CREATE INDEX idx_jwks_expires_at ON jwks(expires_at);
CREATE INDEX idx_jwks_created_at ON jwks(created_at);