-- Per-member permission overrides for MySQL

CREATE TABLE IF NOT EXISTS member_permission_override (
    id              VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL,
    user_id         VARCHAR(255) NOT NULL,
    permission      VARCHAR(255) NOT NULL,
    effect          VARCHAR(10) NOT NULL,
    created_at      VARCHAR(255) NOT NULL,
    updated_at      VARCHAR(255) NOT NULL,
    UNIQUE (organization_id, user_id, permission),
    FOREIGN KEY (organization_id) REFERENCES organization(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES `user`(id) ON DELETE CASCADE
);

CREATE INDEX idx_member_permission_override_org_user
    ON member_permission_override(organization_id, user_id);
