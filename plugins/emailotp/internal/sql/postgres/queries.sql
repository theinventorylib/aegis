-- Email OTP User queries

-- name: CreateUser :exec
INSERT INTO "user" (id, avatar, name, email, created_at, updated_at, disabled, email_verified)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetUserByID :one
SELECT u.id, u.avatar, u.name, u.email, u.created_at, u.updated_at, u.disabled,
       u.email, u.email_verified
FROM "user" u
WHERE u.id = $1;

-- name: GetUserByEmail :one
SELECT u.id, u.avatar, u.name, u.email, u.created_at, u.updated_at, u.disabled,
       u.email, u.email_verified
FROM "user" u
WHERE u.email = $1;

-- name: UpdateUserEmail :exec
UPDATE "user"
SET email = $2, email_verified = $3, updated_at = $4
WHERE id = $1;