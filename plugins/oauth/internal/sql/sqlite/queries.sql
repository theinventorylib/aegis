-- OAuth Connection queries

-- name: CreateConnection :exec
INSERT INTO oauth_connection (id, user_id, provider, provider_user_id, email, name, avatar_url, access_token, refresh_token, expires_at, provider_data, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetConnectionByProviderUserID :one
SELECT id, user_id, provider, provider_user_id, email, name, avatar_url, access_token, refresh_token, expires_at, provider_data, created_at, updated_at
FROM oauth_connection
WHERE provider = ? AND provider_user_id = ?;

-- name: GetConnectionsByUserID :many
SELECT id, user_id, provider, provider_user_id, email, name, avatar_url, access_token, refresh_token, expires_at, provider_data, created_at, updated_at
FROM oauth_connection
WHERE user_id = ?;

-- name: UpdateConnection :exec
UPDATE oauth_connection
SET user_id = ?, provider = ?, provider_user_id = ?, email = ?, name = ?, avatar_url = ?, access_token = ?, refresh_token = ?, expires_at = ?, provider_data = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteConnection :exec
DELETE FROM oauth_connection WHERE provider = ? AND user_id = ?;
