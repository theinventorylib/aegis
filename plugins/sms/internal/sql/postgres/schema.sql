-- SMS plugin schema for PostgreSQL

-- Add phone fields to user table
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS phone_number TEXT;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS phone_verified INTEGER NOT NULL DEFAULT 0;

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_user_phone ON "user"(phone_number);
CREATE INDEX IF NOT EXISTS idx_user_phone_verified ON "user"(phone_verified);