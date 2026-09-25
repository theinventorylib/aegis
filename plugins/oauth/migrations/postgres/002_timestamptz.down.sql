-- Revert OAuth connection timestamps to RFC3339 TEXT.
ALTER TABLE oauth_connection
    ALTER COLUMN expires_at TYPE TEXT USING expires_at::text,
    ALTER COLUMN created_at TYPE TEXT USING created_at::text,
    ALTER COLUMN updated_at TYPE TEXT USING updated_at::text;
