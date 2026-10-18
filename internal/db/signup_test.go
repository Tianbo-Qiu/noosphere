package db

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreatePasswordIdentity", func() {
	It("creates the identity, its password credential and the identifier", func(ctx SpecContext) {
		tx := testTx(ctx)
		traits := Traits{Email: "test@example.com"}
		config := PasswordConfig{HashedPassword: "test-hash"}

		identity, err := CreatePasswordIdentity(ctx, tx, traits, "test@example.com", config)
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

	DescribeTable("stores the normalized identifier and keeps the traits as typed",
		func(ctx SpecContext, typed, want string) {
			tx := testTx(ctx)

			identity, err := CreatePasswordIdentity(ctx, tx, Traits{Email: typed}, typed, PasswordConfig{HashedPassword: "test-hash"})
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
			_, err := CreatePasswordIdentity(ctx, tx, Traits{Email: first}, first, config)
			Expect(err).NotTo(HaveOccurred())

			_, err = CreatePasswordIdentity(ctx, tx, Traits{Email: second}, second, config)
			Expect(err).To(MatchError(ErrIdentifierTaken))

			var count int
			Expect(tx.QueryRow(ctx, `SELECT count(*) FROM identities`).Scan(&count)).To(Succeed())
			Expect(count).To(Equal(1))
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

			_, err := CreatePasswordIdentity(ctx, tx, Traits{Email: rawIdentifier}, rawIdentifier, PasswordConfig{HashedPassword: "test-hash"})
			Expect(err).To(MatchError(ErrInvalidIdentifier))

			var count int
			Expect(tx.QueryRow(ctx, `SELECT count(*) FROM identities`).Scan(&count)).To(Succeed())
			Expect(count).To(Equal(0))
		},
		Entry("zero-width space", "test\u200b@example.com"), // zero-width space after "test"
		Entry("empty", ""),
		Entry("whitespace only", " \t\n"),
	)
})
