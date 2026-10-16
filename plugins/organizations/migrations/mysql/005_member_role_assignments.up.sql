-- Scoped organization-role assignments for MySQL
--
-- `members.role` remains the member's organization-wide (unscoped) role;
-- this table stores additional assignments that are limited to a scope.
-- Both scope fields are opaque app-defined strings (e.g. scope_type
-- 'region', scope_id 'region_123'); Aegis only stores and compares them.
--
-- The unique key makes assignment idempotent (the same role at the same scope
-- cannot pile up), and its (organization_id, user_id) prefix serves per-member
-- lookups. The string columns are sized so that key fits InnoDB's 3072-byte
-- index limit under utf8mb4 (5 columns x 4 bytes/char): ids are ULIDs and
-- role names are API-capped at 50 chars, scope type/id at 128 (enforced by
-- AssignRole). MySQL accepts foreign-key columns shorter than the referenced
-- string column, so organization_id/user_id stay narrower than
-- organization(id)/user(id).

CREATE TABLE IF NOT EXISTS member_role_assignment (
    id              VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(64) NOT NULL,
    user_id         VARCHAR(64) NOT NULL,
    role            VARCHAR(128) NOT NULL,
    scope_type      VARCHAR(128) NOT NULL,
    scope_id        VARCHAR(128) NOT NULL,
    created_at      VARCHAR(255) NOT NULL,
    updated_at      VARCHAR(255) NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organization(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES `user`(id) ON DELETE CASCADE,
    UNIQUE (organization_id, user_id, role, scope_type, scope_id)
);

CREATE INDEX idx_member_role_assignment_role ON member_role_assignment(organization_id, role);
