-- Invitation table for MySQL
-- Tracks pending invitations to join organizations and teams.

CREATE TABLE IF NOT EXISTS invitation (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL,
    team_id VARCHAR(255),
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    inviter_id VARCHAR(255) NOT NULL,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    expires_at VARCHAR(255) NOT NULL,
    created_at VARCHAR(255) NOT NULL,
    updated_at VARCHAR(255) NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organization(id) ON DELETE CASCADE,
    FOREIGN KEY (team_id) REFERENCES team(id) ON DELETE CASCADE,
    FOREIGN KEY (inviter_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX idx_invitation_org_status ON invitation(organization_id, status);
CREATE INDEX idx_invitation_token_hash ON invitation(token_hash);
CREATE INDEX idx_invitation_status_expires ON invitation(status, expires_at);
