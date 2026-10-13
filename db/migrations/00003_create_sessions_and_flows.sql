-- +goose Up
CREATE TABLE sessions (
    id               uuid        PRIMARY KEY DEFAULT uuidv7(),
    token_hash       bytea       NOT NULL UNIQUE,
    identity_id      uuid        NOT NULL REFERENCES identities (id) ON DELETE CASCADE,
    active           boolean     NOT NULL DEFAULT true,
    authenticated_at timestamptz NOT NULL DEFAULT now(),
    issued_at        timestamptz NOT NULL DEFAULT now(),
    expires_at       timestamptz NOT NULL
);

CREATE INDEX sessions_identity_id_idx ON sessions (identity_id);

CREATE TABLE registration_flows (
    id         uuid        PRIMARY KEY DEFAULT uuidv7(),
    csrf_token text        NOT NULL,
    state      jsonb       NOT NULL DEFAULT '{}',
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE registration_flows;
DROP TABLE sessions;
