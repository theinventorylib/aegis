package email

import "github.com/theinventorylib/aegis/plugins"

// GetMigrations returns database migrations for email verification
func (p *Plugin) GetMigrations() []plugins.Migration {
	return []plugins.Migration{
		{
			Version:     "001",
			Description: "Add email columns to auth.user table",
			Up: `
				ALTER TABLE auth.user 
				ADD COLUMN IF NOT EXISTS email VARCHAR(255),
				ADD COLUMN IF NOT EXISTS email_verified BOOLEAN DEFAULT FALSE;
				CREATE INDEX IF NOT EXISTS idx_user_email ON auth.user(email);
			`,
			Down: `
				DROP INDEX IF EXISTS idx_user_email;
				ALTER TABLE auth.user DROP COLUMN IF EXISTS email_verified;
				ALTER TABLE auth.user DROP COLUMN IF EXISTS email;
			`,
		},
	}
}
