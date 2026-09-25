-- Revert core timestamp columns to RFC3339 TEXT.
--
-- timestamptz::text renders using the session DateStyle/TimeZone (for example
-- "2026-09-25 18:00:00+02"), which is not RFC3339 and would break the
-- pre-TIMESTAMPTZ code that parses these values. Render UTC RFC3339 explicitly.
ALTER TABLE "user"
    ALTER COLUMN created_at TYPE TEXT USING to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
    ALTER COLUMN updated_at TYPE TEXT USING to_char(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"');

ALTER TABLE accounts
    ALTER COLUMN expires_at TYPE TEXT USING to_char(expires_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
    ALTER COLUMN created_at TYPE TEXT USING to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
    ALTER COLUMN updated_at TYPE TEXT USING to_char(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"');

ALTER TABLE verification
    ALTER COLUMN expires_at TYPE TEXT USING to_char(expires_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
    ALTER COLUMN created_at TYPE TEXT USING to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"');

ALTER TABLE session
    ALTER COLUMN expires_at TYPE TEXT USING to_char(expires_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
    ALTER COLUMN created_at TYPE TEXT USING to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"');
