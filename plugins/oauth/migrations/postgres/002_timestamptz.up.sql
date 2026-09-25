-- Convert OAuth connection timestamps from RFC3339 TEXT to TIMESTAMPTZ.
-- No-op for databases created after 001 was updated to TIMESTAMPTZ.
ALTER TABLE oauth_connection
    ALTER COLUMN expires_at TYPE TIMESTAMPTZ USING expires_at::timestamptz,
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at::timestamptz,
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at::timestamptz;
