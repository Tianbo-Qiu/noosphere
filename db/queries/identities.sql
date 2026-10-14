-- name: CreateIdentity :one
INSERT INTO identities (traits)
VALUES ($1)
RETURNING *;
