package jwt

import "github.com/theinventorylib/aegis/plugins"

// GetMigrations returns plugin migrations
func (p *Plugin) GetMigrations() []plugins.Migration {
	return []plugins.Migration{
		{
			Version:     "001",
			Description: "Create JWT schema",
			Up: `
                -- Create jwt schema
                CREATE SCHEMA IF NOT EXISTS jwt;

                -- JWT Keys Table
                CREATE TABLE jwt.jwks (
                    kid TEXT PRIMARY KEY,
                    key_data JSONB NOT NULL,
                    algorithm TEXT NOT NULL,
                    use TEXT DEFAULT 'sig', -- 'sig' for signing, 'enc' for encryption
                    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
                    expires_at TIMESTAMP
                );

                CREATE INDEX idx_jwt_jwks_algorithm ON jwt.jwks(algorithm);
                CREATE INDEX idx_jwt_jwks_use ON jwt.jwks(use);
                CREATE INDEX idx_jwt_jwks_expires_at ON jwt.jwks(expires_at);
            `,
			Down: `
                DROP TABLE IF EXISTS jwt.jwks;
                DROP SCHEMA IF EXISTS jwt CASCADE;
            `,
		},
	}
}
