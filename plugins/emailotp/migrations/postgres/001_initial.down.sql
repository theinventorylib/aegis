-- Reverse email OTP plugin schema changes
DROP INDEX IF EXISTS idx_user_email_verified;

ALTER TABLE "user" DROP COLUMN IF EXISTS email_verified;