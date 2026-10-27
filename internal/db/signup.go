package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Tianbo-Qiu/noosphere/internal/identifier"
	"github.com/jackc/pgx/v5"
)

const identifierUniqueConstraint = "identity_credential_identifiers_type_identifier_key"

var (
	ErrIdentifierTaken   = errors.New("identifier already taken")
	ErrInvalidIdentifier = errors.New("invalid identifier")
)

type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// SignUpWithPassword creates an identity with a password credential and its
// identifier, and signs it in with a new session, in one transaction. Either
// all of them are stored or none.
func SignUpWithPassword(ctx context.Context, db Beginner, traits Traits, rawIdentifier string, config PasswordConfig, sessionTokenHash []byte, sessionExpiresAt time.Time) (Identity, Session, error) {
	normalized, err := identifier.Normalize(rawIdentifier)
	if err != nil {
		return Identity{}, Session{}, fmt.Errorf("%w: %w", ErrInvalidIdentifier, err)
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return Identity{}, Session{}, err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return Identity{}, Session{}, err
	}
	defer tx.Rollback(ctx)

	q := New(tx)
	identity, err := q.CreateIdentity(ctx, traits)
	if err != nil {
		return Identity{}, Session{}, err
	}

	credential, err := q.CreateCredential(ctx, CreateCredentialParams{
		IdentityID: identity.ID,
		Type:       CredentialTypePassword,
		Config:     configJSON,
	})
	if err != nil {
		return Identity{}, Session{}, err
	}

	_, err = q.CreateCredentialIdentifier(ctx, CreateCredentialIdentifierParams{
		CredentialID: credential.ID,
		Type:         CredentialTypePassword,
		Identifier:   normalized,
	})
	if err != nil {
		if isConstraintViolation(err, uniqueViolation, identifierUniqueConstraint) {
			return Identity{}, Session{}, ErrIdentifierTaken
		}
		return Identity{}, Session{}, err
	}

	session, err := q.CreateSession(ctx, CreateSessionParams{
		TokenHash:  sessionTokenHash,
		IdentityID: identity.ID,
		ExpiresAt:  sessionExpiresAt,
	})
	if err != nil {
		return Identity{}, Session{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Identity{}, Session{}, err
	}

	return identity, session, nil
}
