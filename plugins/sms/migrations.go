package sms

import "github.com/theinventorylib/aegis/plugins"

// GetMigrations returns database migrations for SMS verification
func (p *Plugin) GetMigrations() []plugins.Migration {
	return []plugins.Migration{
		{
			Version:     "001",
			Description: "Add phone_number column to auth.user table",
			Up: `
				ALTER TABLE auth.user 
				ADD COLUMN IF NOT EXISTS phone_number VARCHAR(50),
				ADD COLUMN IF NOT EXISTS phone_verified BOOLEAN DEFAULT FALSE;
				CREATE INDEX IF NOT EXISTS idx_user_phone ON auth.user(phone_number);
			`,
			Down: `
				DROP INDEX IF EXISTS idx_user_phone;
				ALTER TABLE auth.user DROP COLUMN IF EXISTS phone_verified;
				ALTER TABLE auth.user DROP COLUMN IF EXISTS phone_number;
			`,
		},
	}
}
