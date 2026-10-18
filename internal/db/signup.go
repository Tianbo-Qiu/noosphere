package db

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

const identifierUniqueConstraint = "identity_credential_identifiers_type_identifier_key"

var ErrIdentifierTaken = errors.New("identifier already taken")

type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

func CreatePasswordIdentity(ctx context.Context, db Beginner, traits Traits, identifier string, config PasswordConfig) (Identity, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return Identity{}, err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return Identity{}, err
	}
	defer tx.Rollback(ctx)

	q := New(tx)
	identity, err := q.CreateIdentity(ctx, traits)
	if err != nil {
		return Identity{}, err
	}

	credential, err := q.CreateCredential(ctx, CreateCredentialParams{
		IdentityID: identity.ID,
		Type:       CredentialTypePassword,
		Config:     configJSON,
	})
	if err != nil {
		return Identity{}, err
	}

	_, err = q.CreateCredentialIdentifier(ctx, CreateCredentialIdentifierParams{
		CredentialID: credential.ID,
		Type:         CredentialTypePassword,
		Identifier:   identifier,
	})
	if err != nil {
		if isConstraintViolation(err, uniqueViolation, identifierUniqueConstraint) {
			return Identity{}, ErrIdentifierTaken
		}
		return Identity{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Identity{}, err
	}

	return identity, nil
}
