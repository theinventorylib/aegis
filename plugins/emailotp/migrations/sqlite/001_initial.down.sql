DROP INDEX IF EXISTS idx_user_email_verified;
ALTER TABLE "user" DROP COLUMN email_verified;