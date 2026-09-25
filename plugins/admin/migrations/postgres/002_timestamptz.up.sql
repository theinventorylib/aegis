-- Convert ban_expiry from RFC3339 TEXT to TIMESTAMPTZ.
-- No-op for fresh installs: 001 already creates the column as TIMESTAMPTZ.
ALTER TABLE "user"
    ALTER COLUMN ban_expiry TYPE TIMESTAMPTZ USING ban_expiry::timestamptz;
