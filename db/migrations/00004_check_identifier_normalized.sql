-- +goose Up
ALTER TABLE identity_credential_identifiers
ADD CONSTRAINT identity_credential_identifiers_identifier_normalized
CHECK (
    identifier = lower(identifier)
    AND identifier = btrim(identifier)
    AND identifier IS NFC NORMALIZED
    AND identifier <> ''
);

-- +goose Down
ALTER TABLE identity_credential_identifiers
DROP CONSTRAINT identity_credential_identifiers_identifier_normalized;
