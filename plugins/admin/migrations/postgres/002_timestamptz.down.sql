-- Revert ban_expiry to RFC3339 TEXT.
ALTER TABLE "user"
    ALTER COLUMN ban_expiry TYPE TEXT USING ban_expiry::text;
