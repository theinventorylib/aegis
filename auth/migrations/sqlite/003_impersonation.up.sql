CREATE TABLE IF NOT EXISTS session_impersonation (
    session_id TEXT PRIMARY KEY,
    actor_user_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    FOREIGN KEY (session_id) REFERENCES session(id) ON DELETE CASCADE,
    FOREIGN KEY (actor_user_id) REFERENCES "user"(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_session_impersonation_actor ON session_impersonation(actor_user_id);
CREATE INDEX IF NOT EXISTS idx_session_impersonation_org ON session_impersonation(organization_id);
CREATE INDEX IF NOT EXISTS idx_session_impersonation_expires ON session_impersonation(expires_at);
