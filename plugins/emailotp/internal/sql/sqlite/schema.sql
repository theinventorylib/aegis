-- Email OTP plugin schema for SQLite

-- Add email fields to user table
ALTER TABLE "user" ADD COLUMN email_verified INTEGER NOT NULL DEFAULT 0;

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_user_email ON "user"(email);
CREATE INDEX IF NOT EXISTS idx_user_email_verified ON "user"(email_verified);
