-- Convert ban_expiry from RFC3339 TEXT to TIMESTAMPTZ.
-- No-op for databases created after 001 was updated to TIMESTAMPTZ.
ALTER TABLE "user"
    ALTER COLUMN ban_expiry TYPE TIMESTAMPTZ USING ban_expiry::timestamptz;
