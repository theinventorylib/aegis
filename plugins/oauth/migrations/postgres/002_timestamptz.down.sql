-- Revert OAuth connection timestamps to RFC3339 TEXT.
-- See auth 002_timestamptz.down.sql for why to_char is used instead of ::text.
ALTER TABLE oauth_connection
    ALTER COLUMN expires_at TYPE TEXT USING to_char(expires_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
    ALTER COLUMN created_at TYPE TEXT USING to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
    ALTER COLUMN updated_at TYPE TEXT USING to_char(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"');
