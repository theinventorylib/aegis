DROP INDEX IF EXISTS `idx_user_email_verified` ON `user`;
ALTER TABLE `user` DROP COLUMN IF EXISTS `email_verified`;