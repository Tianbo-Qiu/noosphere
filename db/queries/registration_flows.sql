-- name: CreateRegistrationFlow :one
INSERT INTO registration_flows (csrf_token, expires_at)
VALUES ($1, $2)
RETURNING *;

-- name: GetRegistrationFlow :one
SELECT * FROM registration_flows
WHERE id = $1;
