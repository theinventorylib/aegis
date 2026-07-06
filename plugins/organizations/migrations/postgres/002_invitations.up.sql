-- Invitation table for PostgreSQL
-- Tracks pending invitations to join organizations and teams.

CREATE TABLE IF NOT EXISTS invitation (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    team_id TEXT,
    email TEXT NOT NULL,
    role TEXT NOT NULL,
    inviter_id TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending',
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organization(id) ON DELETE CASCADE,
    FOREIGN KEY (team_id) REFERENCES team(id) ON DELETE CASCADE,
    FOREIGN KEY (inviter_id) REFERENCES "user"(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_invitation_org_status ON invitation(organization_id, status);
CREATE INDEX IF NOT EXISTS idx_invitation_token_hash ON invitation(token_hash);
CREATE INDEX IF NOT EXISTS idx_invitation_status_expires ON invitation(status, expires_at);
