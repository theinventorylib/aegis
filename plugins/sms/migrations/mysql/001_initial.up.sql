-- SMS plugin schema for MySQL

-- Add phone fields to user table
ALTER TABLE `user` ADD COLUMN phone_number VARCHAR(20);
ALTER TABLE `user` ADD COLUMN phone_verified TINYINT NOT NULL DEFAULT 0;

-- Create indexes
CREATE INDEX idx_user_phone ON `user`(phone_number);
CREATE INDEX idx_user_phone_verified ON `user`(phone_verified);
