CREATE TABLE IF NOT EXISTS session_impersonation (
    session_id VARCHAR(255) PRIMARY KEY,
    actor_user_id VARCHAR(255) NOT NULL,
    organization_id VARCHAR(255) NOT NULL,
    created_at VARCHAR(255) NOT NULL,
    expires_at VARCHAR(255) NOT NULL,
    KEY idx_session_impersonation_actor (actor_user_id),
    KEY idx_session_impersonation_org (organization_id),
    KEY idx_session_impersonation_expires (expires_at),
    CONSTRAINT fk_session_impersonation_session FOREIGN KEY (session_id) REFERENCES session(id) ON DELETE CASCADE,
    CONSTRAINT fk_session_impersonation_actor FOREIGN KEY (actor_user_id) REFERENCES `user`(id) ON DELETE CASCADE
);
