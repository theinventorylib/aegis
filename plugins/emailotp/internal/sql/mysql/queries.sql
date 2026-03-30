-- Email OTP User queries

-- name: CreateUser :exec
INSERT INTO `user` (id, avatar, name, email, created_at, updated_at, disabled, email_verified)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetUserByID :one
SELECT u.id, u.avatar, u.name, u.email, u.created_at, u.updated_at, u.disabled,
       u.email, u.email_verified
FROM `user` u
WHERE u.id = ?;

-- name: GetUserByEmail :one
SELECT u.id, u.avatar, u.name, u.email, u.created_at, u.updated_at, u.disabled,
       u.email, u.email_verified
FROM `user` u
WHERE u.email = ?;

-- name: UpdateUserEmail :exec
UPDATE `user`
SET email = ?, email_verified = ?, updated_at = ?
WHERE id = ?;
