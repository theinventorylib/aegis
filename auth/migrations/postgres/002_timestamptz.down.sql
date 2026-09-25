-- Revert core timestamp columns to RFC3339 TEXT.
ALTER TABLE "user"
    ALTER COLUMN created_at TYPE TEXT USING created_at::text,
    ALTER COLUMN updated_at TYPE TEXT USING updated_at::text;

ALTER TABLE accounts
    ALTER COLUMN expires_at TYPE TEXT USING expires_at::text,
    ALTER COLUMN created_at TYPE TEXT USING created_at::text,
    ALTER COLUMN updated_at TYPE TEXT USING updated_at::text;

ALTER TABLE verification
    ALTER COLUMN expires_at TYPE TEXT USING expires_at::text,
    ALTER COLUMN created_at TYPE TEXT USING created_at::text;

ALTER TABLE session
    ALTER COLUMN expires_at TYPE TEXT USING expires_at::text,
    ALTER COLUMN created_at TYPE TEXT USING created_at::text;
