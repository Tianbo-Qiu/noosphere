package db

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateCredential", func() {
	It("stores a password credential for the identity", func(ctx SpecContext) {
		q := New(testTx(ctx))
		identity := newTestIdentity(ctx, q, "test@example.com")

		got, err := q.CreateCredential(ctx, CreateCredentialParams{
			IdentityID: identity.ID,
			Type:       CredentialTypePassword,
			Config:     json.RawMessage(`{"hashed_password": "test-hash"}`),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(got.ID.Version()).To(BeEquivalentTo(7))
		Expect(got.IdentityID).To(Equal(identity.ID))
		Expect(got.Type).To(Equal(CredentialTypePassword))
		Expect(got.Config).To(MatchJSON(`{"hashed_password": "test-hash"}`))
	})
})

var _ = Describe("CreateCredentialIdentifier", func() {
	It("stores the identifier for the credential", func(ctx SpecContext) {
		q := New(testTx(ctx))
		credential := newTestPasswordCredential(ctx, q, "test@example.com")

		got, err := q.CreateCredentialIdentifier(ctx, CreateCredentialIdentifierParams{
			CredentialID: credential.ID,
			Type:         credential.Type,
			Identifier:   "test@example.com",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(got.ID.Version()).To(BeEquivalentTo(7))
		Expect(got.CredentialID).To(Equal(credential.ID))
		Expect(got.Type).To(Equal(CredentialTypePassword))
		Expect(got.Identifier).To(Equal("test@example.com"))
	})

	It("rejects an identifier already used by another credential of the same type", func(ctx SpecContext) {
		q := New(testTx(ctx))
		first := newTestPasswordCredential(ctx, q, "test@example.com")
		second := newTestPasswordCredential(ctx, q, "test@example.com")
		newTestCredentialIdentifier(ctx, q, first, "test@example.com")

		_, err := q.CreateCredentialIdentifier(ctx, CreateCredentialIdentifierParams{
			CredentialID: second.ID,
			Type:         second.Type,
			Identifier:   "test@example.com",
		})

		expectPgError(err, uniqueViolation, "identity_credential_identifiers_type_identifier_key")
	})

	It("rejects an identifier whose type differs from its credential's", func(ctx SpecContext) {
		q := New(testTx(ctx))
		credential := newTestPasswordCredential(ctx, q, "test@example.com")

		_, err := q.CreateCredentialIdentifier(ctx, CreateCredentialIdentifierParams{
			CredentialID: credential.ID,
			Type:         "code",
			Identifier:   "test@example.com",
		})

		expectPgError(err, foreignKeyViolation, "identity_credential_identifiers_credential_id_type_fkey")
	})

	DescribeTable("rejects an identifier that is not normalized",
		func(ctx SpecContext, identifier string) {
			q := New(testTx(ctx))
			credential := newTestPasswordCredential(ctx, q, "test@example.com")

			_, err := q.CreateCredentialIdentifier(ctx, CreateCredentialIdentifierParams{
				CredentialID: credential.ID,
				Type:         credential.Type,
				Identifier:   identifier,
			})

			expectPgError(err, checkViolation, "identity_credential_identifiers_identifier_normalized")
		},
		Entry("uppercase", "Test@Example.com"),
		Entry("leading space", " test@example.com"),
		Entry("trailing space", "test@example.com "),
		Entry("decomposed é", "cafe\u0301@example.com"), // e + combining acute (2 code points)
		Entry("empty", ""),
	)
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
		Type:       CredentialTypePassword,
		Config:     json.RawMessage(`{"hashed_password": "test-hash"}`),
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
