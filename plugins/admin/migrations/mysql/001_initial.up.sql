-- Admin plugin schema for MySQL
ALTER TABLE `user` ADD COLUMN role VARCHAR(50) DEFAULT 'user';

-- Add ban management fields
ALTER TABLE `user` ADD COLUMN banned TINYINT NOT NULL DEFAULT 0;
ALTER TABLE `user` ADD COLUMN ban_reason TEXT;
ALTER TABLE `user` ADD COLUMN ban_expiry VARCHAR(255);
ALTER TABLE `user` ADD COLUMN ban_counter INT NOT NULL DEFAULT 0;

-- Create indexes
CREATE INDEX idx_user_role ON `user`(role);
CREATE INDEX idx_user_banned ON `user`(banned);
CREATE INDEX idx_user_ban_expiry ON `user`(ban_expiry);
