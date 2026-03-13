-- Reverse SMS plugin schema changes
DROP INDEX IF EXISTS idx_user_phone_verified;
DROP INDEX IF EXISTS idx_user_phone;

ALTER TABLE "user" DROP COLUMN IF EXISTS phone_verified;
ALTER TABLE "user" DROP COLUMN IF EXISTS phone_number;