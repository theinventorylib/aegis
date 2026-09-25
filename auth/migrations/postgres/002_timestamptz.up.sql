-- Convert core timestamp columns from RFC3339 TEXT to TIMESTAMPTZ.
--
-- Fresh installs already create TIMESTAMPTZ columns (001), so this is a no-op
-- for them. Databases created before that change still hold TEXT; the USING
-- cast interprets each value, and Aegis-written values carry an explicit
-- offset so they convert correctly regardless of the session timezone.

ALTER TABLE "user"
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at::timestamptz,
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at::timestamptz;

ALTER TABLE accounts
    ALTER COLUMN expires_at TYPE TIMESTAMPTZ USING expires_at::timestamptz,
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at::timestamptz,
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at::timestamptz;

ALTER TABLE verification
    ALTER COLUMN expires_at TYPE TIMESTAMPTZ USING expires_at::timestamptz,
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at::timestamptz;

ALTER TABLE session
    ALTER COLUMN expires_at TYPE TIMESTAMPTZ USING expires_at::timestamptz,
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at::timestamptz;
