-- Revert ban_expiry to RFC3339 TEXT. See auth 002_timestamptz.down.sql for why
-- to_char is used instead of a plain ::text cast.
ALTER TABLE "user"
    ALTER COLUMN ban_expiry TYPE TEXT USING to_char(ban_expiry AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"');
