CREATE TABLE IF NOT EXISTS session_impersonation (
    session_id VARCHAR(255) PRIMARY KEY,
    actor_user_id VARCHAR(255) NOT NULL,
    organization_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (session_id) REFERENCES session(id) ON DELETE CASCADE,
    FOREIGN KEY (actor_user_id) REFERENCES "user"(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_session_impersonation_actor ON session_impersonation(actor_user_id);
CREATE INDEX IF NOT EXISTS idx_session_impersonation_org ON session_impersonation(organization_id);
CREATE INDEX IF NOT EXISTS idx_session_impersonation_expires ON session_impersonation(expires_at);
