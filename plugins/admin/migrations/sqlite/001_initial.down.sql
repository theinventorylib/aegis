-- Reverse admin plugin schema changes
DROP INDEX IF EXISTS idx_user_ban_expiry;
DROP INDEX IF EXISTS idx_user_banned;
DROP INDEX IF EXISTS idx_user_role;

ALTER TABLE "user" DROP COLUMN ban_counter;
ALTER TABLE "user" DROP COLUMN ban_expiry;
ALTER TABLE "user" DROP COLUMN ban_reason;
ALTER TABLE "user" DROP COLUMN banned;
ALTER TABLE "user" DROP COLUMN role;