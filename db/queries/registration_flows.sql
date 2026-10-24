-- name: CreateRegistrationFlow :one
INSERT INTO registration_flows (csrf_token, expires_at)
VALUES ($1, $2)
RETURNING *;

-- name: GetRegistrationFlow :one
SELECT * FROM registration_flows
WHERE id = $1;

-- name: UpdateRegistrationFlowState :exec
UPDATE registration_flows
SET state = $2, updated_at = now()
WHERE id = $1;

-- name: DeleteRegistrationFlow :exec
DELETE FROM registration_flows
WHERE id = $1;
