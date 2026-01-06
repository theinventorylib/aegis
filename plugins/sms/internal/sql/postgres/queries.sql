-- SMS User Phone queries

-- name: CreateUser :exec
INSERT INTO "user" (id, avatar, name, email, created_at, updated_at, disabled, phone_number, phone_verified)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetUserByID :one
SELECT u.id, u.avatar, u.name, u.email, u.created_at, u.updated_at, u.disabled,
       u.phone_number, u.phone_verified
FROM "user" u
WHERE u.id = $1;

-- name: GetUserByPhone :one
SELECT u.id, u.avatar, u.name, u.email, u.created_at, u.updated_at, u.disabled,
       u.phone_number, u.phone_verified
FROM "user" u
WHERE u.phone_number = $1;

-- name: UpdateUserPhone :exec
UPDATE "user"
SET phone_number = $2, phone_verified = $3, updated_at = $4
WHERE id = $1;
