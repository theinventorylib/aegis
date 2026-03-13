-- Reverse admin plugin schema changes
DROP INDEX IF EXISTS idx_user_ban_expiry;
DROP INDEX IF EXISTS idx_user_banned;
DROP INDEX IF EXISTS idx_user_role;

ALTER TABLE "user" DROP COLUMN IF EXISTS ban_counter;
ALTER TABLE "user" DROP COLUMN IF EXISTS ban_expiry;
ALTER TABLE "user" DROP COLUMN IF EXISTS ban_reason;
ALTER TABLE "user" DROP COLUMN IF EXISTS banned;
ALTER TABLE "user" DROP COLUMN IF EXISTS role;