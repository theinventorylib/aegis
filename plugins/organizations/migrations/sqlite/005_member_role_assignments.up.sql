-- Scoped organization-role assignments for SQLite
--
-- `members.role` remains the member's organization-wide (unscoped) role;
-- this table stores additional assignments that are limited to a scope.
-- Both scope fields are opaque app-defined strings (e.g. scope_type
-- 'region', scope_id 'region_123'); Aegis only stores and compares them.
--
-- The unique key makes assignment idempotent (the same role at the same scope
-- cannot pile up), and its (organization_id, user_id) prefix serves per-member
-- lookups.

CREATE TABLE IF NOT EXISTS member_role_assignment (
    id              TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    user_id         TEXT NOT NULL,
    role            TEXT NOT NULL,
    scope_type      TEXT NOT NULL,
    scope_id        TEXT NOT NULL,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organization(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE,
    UNIQUE (organization_id, user_id, role, scope_type, scope_id)
);

CREATE INDEX IF NOT EXISTS idx_member_role_assignment_role
    ON member_role_assignment(organization_id, role);
