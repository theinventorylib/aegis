-- Admin plugin schema for PostgreSQL
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'user';

-- Add ban management fields
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS banned INTEGER NOT NULL DEFAULT 0;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS ban_reason TEXT;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS ban_expiry TEXT;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS ban_counter INTEGER NOT NULL DEFAULT 0;

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_user_role ON "user"(role);
CREATE INDEX IF NOT EXISTS idx_user_banned ON "user"(banned);
CREATE INDEX IF NOT EXISTS idx_user_ban_expiry ON "user"(ban_expiry);
