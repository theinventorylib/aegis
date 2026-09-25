-- Reverse TOTP plugin schema
DROP TABLE IF EXISTS totp_session;
ALTER TABLE "user" DROP COLUMN IF EXISTS totp_enabled;
ALTER TABLE "user" DROP COLUMN IF EXISTS totp_secret;
