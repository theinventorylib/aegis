-- Per-member permission overrides for PostgreSQL
--
-- Overrides adjust what a member's role grants: "deny" removes a permission
-- the role would allow, "grant" adds one the role does not.

CREATE TABLE IF NOT EXISTS member_permission_override (
    id              TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    user_id         TEXT NOT NULL,
    permission      TEXT NOT NULL,
    effect          TEXT NOT NULL CHECK (effect IN ('grant', 'deny')),
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    UNIQUE (organization_id, user_id, permission),
    FOREIGN KEY (organization_id) REFERENCES organization(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_member_permission_override_org_user
    ON member_permission_override(organization_id, user_id);
