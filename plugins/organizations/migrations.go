package organizations

import "github.com/theinventorylib/aegis/plugins"

// GetMigrations returns database migrations for SMS verification
func (p *Plugin) GetMigrations() []plugins.Migration {
	return []plugins.Migration{
		{
			Version:     "001",
			Description: "Add organizations tables",
			Up: `
				-- Organizations table
				CREATE TABLE IF NOT EXISTS auth.organizations (
				    id TEXT PRIMARY KEY,
				    name TEXT NOT NULL,
				    slug TEXT NOT NULL UNIQUE,
				    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
				);

				CREATE INDEX IF NOT EXISTS idx_organizations_slug ON auth.organizations(slug);

				-- Create updated_at trigger for organizations
				CREATE TRIGGER IF NOT EXISTS update_organizations_updated_at
				    BEFORE UPDATE ON auth.organizations
				    FOR EACH ROW
				    EXECUTE FUNCTION auth.update_updated_at_column();

				-- User Organizations (memberships)
				CREATE TABLE IF NOT EXISTS auth.user_organizations (
				    id TEXT PRIMARY KEY,
				    user_id TEXT NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
				    organization_id TEXT NOT NULL REFERENCES auth.organizations(id) ON DELETE CASCADE,
				    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
				    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
				    UNIQUE(user_id, organization_id)
				);

				CREATE INDEX IF NOT EXISTS idx_user_organizations_user_id ON auth.user_organizations(user_id);
				CREATE INDEX IF NOT EXISTS idx_user_organizations_org_id ON auth.user_organizations(organization_id);
				CREATE INDEX IF NOT EXISTS idx_user_organizations_role ON auth.user_organizations(role);

				-- Create updated_at trigger for user_organizations
				CREATE TRIGGER IF NOT EXISTS update_user_organizations_updated_at
				    BEFORE UPDATE ON auth.user_organizations
				    FOR EACH ROW
				    EXECUTE FUNCTION auth.update_updated_at_column();

				-- Teams table
				CREATE TABLE IF NOT EXISTS auth.teams (
				    id TEXT PRIMARY KEY,
				    organization_id TEXT NOT NULL REFERENCES auth.organizations(id) ON DELETE CASCADE,
				    name TEXT NOT NULL,
				    description TEXT,
				    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
				);

				CREATE INDEX IF NOT EXISTS idx_teams_org_id ON auth.teams(organization_id);

				-- Create updated_at trigger for teams
				CREATE TRIGGER IF NOT EXISTS update_teams_updated_at
				    BEFORE UPDATE ON auth.teams
				    FOR EACH ROW
				    EXECUTE FUNCTION auth.update_updated_at_column();

				-- Team Members
				CREATE TABLE IF NOT EXISTS auth.team_members (
				    id TEXT PRIMARY KEY,
				    team_id TEXT NOT NULL REFERENCES auth.teams(id) ON DELETE CASCADE,
				    user_id TEXT NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
				    role TEXT NOT NULL CHECK (role IN ('lead', 'member')),
				    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
				    UNIQUE(team_id, user_id)
				);

				CREATE INDEX IF NOT EXISTS idx_team_members_team_id ON auth.team_members(team_id);
				CREATE INDEX IF NOT EXISTS idx_team_members_user_id ON auth.team_members(user_id);

				-- Create updated_at trigger for team_members
				CREATE TRIGGER IF NOT EXISTS update_team_members_updated_at
				    BEFORE UPDATE ON auth.team_members
				    FOR EACH ROW
				    EXECUTE FUNCTION auth.update_updated_at_column();
			`,
			Down: `
				DROP TABLE IF EXISTS auth.team_members;
				DROP TABLE IF EXISTS auth.teams;
				DROP TABLE IF EXISTS auth.user_organizations;
				DROP TABLE IF EXISTS auth.organizations;
			`,
		},
	}
}
