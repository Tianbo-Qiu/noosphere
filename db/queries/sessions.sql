-- name: CreateSession :one
INSERT INTO sessions (token_hash, identity_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;
