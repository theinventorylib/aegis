-- OAuth Connection queries

-- name: CreateConnection :exec
INSERT INTO oauth_connection (id, user_id, provider, provider_user_id, email, name, avatar_url, access_token, refresh_token, expires_at, provider_data, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: GetConnectionByProviderUserID :one
SELECT id, user_id, provider, provider_user_id, email, name, avatar_url, access_token, refresh_token, expires_at, provider_data, created_at, updated_at
FROM oauth_connection
WHERE provider = $1 AND provider_user_id = $2;

-- name: GetConnectionsByUserID :many
SELECT id, user_id, provider, provider_user_id, email, name, avatar_url, access_token, refresh_token, expires_at, provider_data, created_at, updated_at
FROM oauth_connection
WHERE user_id = $1;

-- name: UpdateConnection :exec
UPDATE oauth_connection
SET user_id = $2, provider = $3, provider_user_id = $4, email = $5, name = $6, avatar_url = $7, access_token = $8, refresh_token = $9, expires_at = $10, provider_data = $11, updated_at = $12
WHERE id = $1;

-- name: DeleteConnection :exec
DELETE FROM oauth_connection WHERE provider = $1 AND user_id = $2;
