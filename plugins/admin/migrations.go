package admin

import "github.com/theinventorylib/aegis/plugins"

// GetMigrations returns plugin migrations
func (p *Plugin) GetMigrations() []plugins.Migration {
	return []plugins.Migration{
		{
			Version:     "001",
			Description: "Add RBAC and ban management fields to user table",
			Up: `
-- Add role-based access control field
ALTER TABLE auth.user ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';
ALTER TABLE auth.user ADD CONSTRAINT user_role_check CHECK (role IN ('user', 'admin'));

-- Add ban management fields
ALTER TABLE auth.user ADD COLUMN IF NOT EXISTS banned BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE auth.user ADD COLUMN IF NOT EXISTS ban_reason TEXT;
ALTER TABLE auth.user ADD COLUMN IF NOT EXISTS ban_expiry TIMESTAMP WITH TIME ZONE;
ALTER TABLE auth.user ADD COLUMN IF NOT EXISTS ban_counter INTEGER NOT NULL DEFAULT 0;

-- Create index for role lookups
CREATE INDEX IF NOT EXISTS idx_user_role ON auth.user(role);

-- Create index for ban status
CREATE INDEX IF NOT EXISTS idx_user_banned ON auth.user(banned) WHERE banned = true;

-- Create index for ban expiry (for automatic unbanning)
CREATE INDEX IF NOT EXISTS idx_user_ban_expiry ON auth.user(ban_expiry) WHERE ban_expiry IS NOT NULL;
`,
			Down: `
-- Remove indexes
DROP INDEX IF EXISTS auth.idx_user_ban_expiry;
DROP INDEX IF EXISTS auth.idx_user_banned;
DROP INDEX IF EXISTS auth.idx_user_role;

-- Remove ban management fields
ALTER TABLE auth.user DROP COLUMN IF EXISTS ban_counter;
ALTER TABLE auth.user DROP COLUMN IF EXISTS ban_expiry;
ALTER TABLE auth.user DROP COLUMN IF EXISTS ban_reason;
ALTER TABLE auth.user DROP COLUMN IF EXISTS banned;

-- Remove role field and constraint
ALTER TABLE auth.user DROP CONSTRAINT IF EXISTS user_role_check;
ALTER TABLE auth.user DROP COLUMN IF EXISTS role;
`,
		},
	}
}
