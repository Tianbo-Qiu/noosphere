-- name: CreateCredential :one
INSERT INTO identity_credentials (identity_id, type, config)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateCredentialIdentifier :one
INSERT INTO identity_credential_identifiers (credential_id, type, identifier)
VALUES ($1, $2, $3)
RETURNING *;
