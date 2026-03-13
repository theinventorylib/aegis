-- Email OTP plugin schema for MySQL

-- Add email fields to user table
ALTER TABLE `user` ADD COLUMN IF NOT EXISTS `email_verified` TINYINT NOT NULL DEFAULT 0;

-- Create indexes
CREATE INDEX IF NOT EXISTS `idx_user_email_verified` ON `user`(`email_verified`);