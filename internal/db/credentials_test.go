package db

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testEmail          = "test@example.com"
	testPasswordType   = "password"
	testPasswordConfig = `{"hashed_password": "test-hash"}`
)

var _ = Describe("CreateCredential", func() {
	It("stores a password credential for the identity", func(ctx SpecContext) {
		q := New(testTx(ctx))
		identity := newTestIdentity(ctx, q, testEmail)

		got, err := q.CreateCredential(ctx, CreateCredentialParams{
			IdentityID: identity.ID,
			Type:       testPasswordType,
			Config:     json.RawMessage(testPasswordConfig),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(got.ID.Version()).To(BeEquivalentTo(7))
		Expect(got.IdentityID).To(Equal(identity.ID))
		Expect(got.Type).To(Equal(testPasswordType))
		Expect(got.Config).To(MatchJSON(testPasswordConfig))
	})
})

var _ = Describe("CreateCredentialIdentifier", func() {
	It("stores the identifier for the credential", func(ctx SpecContext) {
		q := New(testTx(ctx))
		credential := newTestPasswordCredential(ctx, q, testEmail)

		got, err := q.CreateCredentialIdentifier(ctx, CreateCredentialIdentifierParams{
			CredentialID: credential.ID,
			Type:         credential.Type,
			Identifier:   testEmail,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(got.ID.Version()).To(BeEquivalentTo(7))
		Expect(got.CredentialID).To(Equal(credential.ID))
		Expect(got.Type).To(Equal(testPasswordType))
		Expect(got.Identifier).To(Equal(testEmail))
	})

	It("rejects an identifier already used by another credential of the same type", func(ctx SpecContext) {
		q := New(testTx(ctx))
		first := newTestPasswordCredential(ctx, q, testEmail)
		second := newTestPasswordCredential(ctx, q, testEmail)
		newTestCredentialIdentifier(ctx, q, first, testEmail)

		_, err := q.CreateCredentialIdentifier(ctx, CreateCredentialIdentifierParams{
			CredentialID: second.ID,
			Type:         second.Type,
			Identifier:   testEmail,
		})

		expectPgError(err, uniqueViolation, "identity_credential_identifiers_type_identifier_key")
	})

	It("rejects an identifier whose type differs from its credential's", func(ctx SpecContext) {
		q := New(testTx(ctx))
		credential := newTestPasswordCredential(ctx, q, testEmail)

		_, err := q.CreateCredentialIdentifier(ctx, CreateCredentialIdentifierParams{
			CredentialID: credential.ID,
			Type:         "code",
			Identifier:   testEmail,
		})

		expectPgError(err, foreignKeyViolation, "identity_credential_identifiers_credential_id_type_fkey")
	})
})

func newTestIdentity(ctx context.Context, q *Queries, email string) Identity {
	GinkgoHelper()
	identity, err := q.CreateIdentity(ctx, Traits{Email: email})
	Expect(err).NotTo(HaveOccurred())
	return identity
}

func newTestPasswordCredential(ctx context.Context, q *Queries, email string) IdentityCredential {
	GinkgoHelper()
	identity := newTestIdentity(ctx, q, email)
	credential, err := q.CreateCredential(ctx, CreateCredentialParams{
		IdentityID: identity.ID,
		Type:       testPasswordType,
		Config:     json.RawMessage(testPasswordConfig),
	})
	Expect(err).NotTo(HaveOccurred())
	return credential
}

func newTestCredentialIdentifier(ctx context.Context, q *Queries, credential IdentityCredential, identifier string) IdentityCredentialIdentifier {
	GinkgoHelper()
	got, err := q.CreateCredentialIdentifier(ctx, CreateCredentialIdentifierParams{
		CredentialID: credential.ID,
		Type:         credential.Type,
		Identifier:   identifier,
	})
	Expect(err).NotTo(HaveOccurred())
	return got
}

func expectPgError(err error, code, constraint string) {
	GinkgoHelper()
	var pgErr *pgconn.PgError
	Expect(errors.As(err, &pgErr)).To(BeTrue(), "err = %v, want *pgconn.PgError", err)
	Expect(pgErr.Code).To(Equal(code))
	Expect(pgErr.ConstraintName).To(Equal(constraint))
}
