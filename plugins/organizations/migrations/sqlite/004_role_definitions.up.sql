-- Persisted custom role definitions for SQLite

CREATE TABLE IF NOT EXISTS organization_role (
    id              TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    name            TEXT NOT NULL,
    permissions     TEXT NOT NULL DEFAULT '[]',
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    UNIQUE (organization_id, name),
    FOREIGN KEY (organization_id) REFERENCES organization(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_organization_role_org ON organization_role(organization_id);
