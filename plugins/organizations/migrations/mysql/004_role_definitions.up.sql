-- Persisted custom role definitions for MySQL

CREATE TABLE IF NOT EXISTS organization_role (
    id              VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    permissions     TEXT NOT NULL,
    created_at      VARCHAR(255) NOT NULL,
    updated_at      VARCHAR(255) NOT NULL,
    UNIQUE (organization_id, name),
    FOREIGN KEY (organization_id) REFERENCES organization(id) ON DELETE CASCADE
);

CREATE INDEX idx_organization_role_org ON organization_role(organization_id);
