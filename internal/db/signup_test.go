package db

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The session a sign-up creates in the specs below.
var (
	testSessionTokenHash = []byte("test-session-token-hash")
	testSessionExpiresAt = time.Now().Add(24 * time.Hour).Truncate(time.Microsecond) // Postgres keeps microseconds
)

var _ = Describe("SignUpWithPassword", func() {
	It("creates the identity, its password credential and the identifier", func(ctx SpecContext) {
		tx := testTx(ctx)
		traits := Traits{Email: "test@example.com"}
		config := PasswordConfig{HashedPassword: "test-hash"}

		identity, _, err := SignUpWithPassword(ctx, tx, traits, "test@example.com", config, testSessionTokenHash, testSessionExpiresAt)
		Expect(err).NotTo(HaveOccurred())
		Expect(identity.Traits).To(Equal(traits))

		var stored PasswordConfig
		Expect(tx.QueryRow(ctx, `
			SELECT c.config
			FROM identity_credentials c
			JOIN identity_credential_identifiers i ON i.credential_id = c.id
			WHERE c.identity_id = $1 AND i.identifier = $2
			`, identity.ID, "test@example.com").Scan(&stored)).To(Succeed())

		Expect(stored).To(Equal(config))
	})

	It("signs the new identity in with an active session holding the token hash", func(ctx SpecContext) {
		tx := testTx(ctx)

		identity, session, err := SignUpWithPassword(ctx, tx, Traits{Email: "test@example.com"}, "test@example.com", PasswordConfig{HashedPassword: "test-hash"}, testSessionTokenHash, testSessionExpiresAt)
		Expect(err).NotTo(HaveOccurred())
		Expect(session.IdentityID).To(Equal(identity.ID))
		Expect(session.TokenHash).To(Equal(testSessionTokenHash))
		Expect(session.Active).To(BeTrue())
		Expect(session.ExpiresAt).To(BeTemporally("==", testSessionExpiresAt))
		Expect(session.AuthenticatedAt).To(BeTemporally("~", time.Now(), time.Minute))

		var stored Session
		Expect(tx.QueryRow(ctx, `SELECT id, identity_id FROM sessions WHERE token_hash = $1`, testSessionTokenHash).
			Scan(&stored.ID, &stored.IdentityID)).To(Succeed())
		Expect(stored.ID).To(Equal(session.ID))
		Expect(stored.IdentityID).To(Equal(identity.ID))
	})

	DescribeTable("stores the normalized identifier and keeps the traits as typed",
		func(ctx SpecContext, typed, want string) {
			tx := testTx(ctx)

			identity, _, err := SignUpWithPassword(ctx, tx, Traits{Email: typed}, typed, PasswordConfig{HashedPassword: "test-hash"}, testSessionTokenHash, testSessionExpiresAt)
			Expect(err).NotTo(HaveOccurred())
			Expect(identity.Traits.Email).To(Equal(typed))

			var stored string
			Expect(tx.QueryRow(ctx, `
				SELECT i.identifier
				FROM identity_credentials c
				JOIN identity_credential_identifiers i ON i.credential_id = c.id
				WHERE c.identity_id = $1
				`, identity.ID).Scan(&stored)).To(Succeed())

			Expect(stored).To(Equal(want))
		},
		Entry("different case", "Test@Example.com", "test@example.com"),
		Entry("uppercase É", "CAF\u00c9@example.com", "caf\u00e9@example.com"),   // É → é
		Entry("decomposed é", "cafe\u0301@example.com", "caf\u00e9@example.com"), // e + combining acute (2 code points) → é (1 code point)
	)

	DescribeTable("returns ErrIdentifierTaken and leaves no new rows when the identifier is taken",
		func(ctx SpecContext, first, second string) {
			tx := testTx(ctx)
			config := PasswordConfig{HashedPassword: "test-hash"}
			_, _, err := SignUpWithPassword(ctx, tx, Traits{Email: first}, first, config, []byte("first-session-token-hash"), testSessionExpiresAt)
			Expect(err).NotTo(HaveOccurred())

			_, _, err = SignUpWithPassword(ctx, tx, Traits{Email: second}, second, config, []byte("second-session-token-hash"), testSessionExpiresAt)
			Expect(err).To(MatchError(ErrIdentifierTaken))

			var identities, sessions int
			Expect(tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM identities), (SELECT count(*) FROM sessions)`).Scan(&identities, &sessions)).To(Succeed())
			Expect(identities).To(Equal(1))
			Expect(sessions).To(Equal(1))
		},
		Entry("same spelling", "test@example.com", "test@example.com"),
		Entry("different case", "test@example.com", "Test@Example.com"),
		Entry("surrounding whitespace", "test@example.com", " test@example.com\n"),
		Entry("fullwidth letters", "test@example.com", "ｔｅｓｔ@example.com"),                    // fullwidth "test"
		Entry("composed and decomposed é", "caf\u00e9@example.com", "cafe\u0301@example.com"), // é (1 code point) vs e + combining acute (2 code points)
	)

	DescribeTable("returns ErrInvalidIdentifier and leaves no rows",
		func(ctx SpecContext, rawIdentifier string) {
			tx := testTx(ctx)

			_, _, err := SignUpWithPassword(ctx, tx, Traits{Email: rawIdentifier}, rawIdentifier, PasswordConfig{HashedPassword: "test-hash"}, testSessionTokenHash, testSessionExpiresAt)
			Expect(err).To(MatchError(ErrInvalidIdentifier))

			var identities, sessions int
			Expect(tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM identities), (SELECT count(*) FROM sessions)`).Scan(&identities, &sessions)).To(Succeed())
			Expect(identities).To(Equal(0))
			Expect(sessions).To(Equal(0))
		},
		Entry("zero-width space", "test\u200b@example.com"), // zero-width space after "test"
		Entry("empty", ""),
		Entry("whitespace only", " \t\n"),
	)
})
