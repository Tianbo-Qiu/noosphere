-- +goose Up
CREATE TABLE identity_credentials (
    id          uuid        PRIMARY KEY DEFAULT uuidv7(),
    identity_id uuid        NOT NULL REFERENCES identities (id) ON DELETE CASCADE,
    type        text        NOT NULL CHECK (type IN ('password')),
    config      jsonb       NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (identity_id, type),
    UNIQUE (id, type)
);

CREATE TABLE identity_credential_identifiers (
    id            uuid PRIMARY KEY DEFAULT uuidv7(),
    credential_id uuid NOT NULL,
    type          text NOT NULL,
    identifier    text NOT NULL,
    UNIQUE (type, identifier),
    FOREIGN KEY (credential_id, type)
        REFERENCES identity_credentials (id, type) ON DELETE CASCADE
);

CREATE INDEX identity_credential_identifiers_credential_id_idx
    ON identity_credential_identifiers (credential_id);

-- +goose Down
DROP TABLE identity_credential_identifiers;
DROP TABLE identity_credentials;
