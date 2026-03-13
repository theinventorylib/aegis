-- OAuth plugin schema for MySQL

-- OAuth connections table
CREATE TABLE IF NOT EXISTS `oauth_connection` (
    `id` TEXT PRIMARY KEY,
    `user_id` TEXT NOT NULL,
    `provider` TEXT NOT NULL,
    `provider_user_id` TEXT NOT NULL,
    `email` TEXT,
    `name` TEXT,
    `avatar_url` TEXT,
    `access_token` TEXT NOT NULL,
    `refresh_token` TEXT,
    `expires_at` TEXT NOT NULL,
    `provider_data` TEXT,
    `created_at` TEXT NOT NULL,
    `updated_at` TEXT NOT NULL,
    FOREIGN KEY (`user_id`) REFERENCES `user`(`id`) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS `idx_oauth_connection_user_id` ON `oauth_connection`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_oauth_connection_provider` ON `oauth_connection`(`provider`);
CREATE INDEX IF NOT EXISTS `idx_oauth_connection_provider_user_id` ON `oauth_connection`(`provider`, `provider_user_id`);