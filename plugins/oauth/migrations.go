package oauth

import "github.com/theinventorylib/aegis/plugins"

// GetMigrations returns the SQL migrations for the OAuth plugin
func (p *Plugin) GetMigrations() []plugins.Migration {
	return []plugins.Migration{
		{
			Version:     "001",
			Description: "Create plugins_oauth schema and connections table",
			Up: `
-- Create OAuth plugin schema
CREATE SCHEMA IF NOT EXISTS plugins_oauth;

-- Create connections table for OAuth provider linkages
CREATE TABLE IF NOT EXISTS plugins_oauth.connections (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES auth.user(id) ON DELETE CASCADE,
    provider TEXT NOT NULL, -- "google", "github", "apple", etc.
    provider_user_id TEXT NOT NULL, -- OAuth provider's user ID
    email TEXT,
    name TEXT,
    avatar_url TEXT,
    access_token TEXT,
    refresh_token TEXT,
    expires_at BIGINT, -- Unix timestamp
    provider_data JSONB DEFAULT '{}', -- Additional provider-specific data
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);

CREATE INDEX IF NOT EXISTS idx_oauth_connections_user_id ON plugins_oauth.connections(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_connections_provider ON plugins_oauth.connections(provider);
CREATE INDEX IF NOT EXISTS idx_oauth_connections_provider_user ON plugins_oauth.connections(provider, provider_user_id);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION plugins_oauth.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_oauth_connections_updated_at
    BEFORE UPDATE ON plugins_oauth.connections
    FOR EACH ROW
    EXECUTE FUNCTION plugins_oauth.update_updated_at_column();
`,
			Down: `
DROP TRIGGER IF EXISTS update_oauth_connections_updated_at ON plugins_oauth.connections;
DROP FUNCTION IF EXISTS plugins_oauth.update_updated_at_column();
DROP TABLE IF EXISTS plugins_oauth.connections;
DROP SCHEMA IF EXISTS plugins_oauth CASCADE;
`,
		},
	}
}
