-- +goose Up
CREATE TABLE identities (
    id         uuid        PRIMARY KEY DEFAULT uuidv7(),
    traits     jsonb       NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE identities;
