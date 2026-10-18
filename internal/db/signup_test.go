package db

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreatePasswordIdentity", func() {
	It("creates the identity, its password credential and the identifier", func(ctx SpecContext) {
		tx := testTx(ctx)
		traits := Traits{Email: testEmail}
		config := PasswordConfig{HashedPassword: testHashedPassword}

		identity, err := CreatePasswordIdentity(ctx, tx, traits, testEmail, config)
		Expect(err).NotTo(HaveOccurred())
		Expect(identity.Traits).To(Equal(traits))

		var stored PasswordConfig
		Expect(tx.QueryRow(ctx, `
			SELECT c.config
			FROM identity_credentials c
			JOIN identity_credential_identifiers i ON i.credential_id = c.id
			WHERE c.identity_id = $1 AND i.identifier = $2
			`, identity.ID, testEmail).Scan(&stored)).To(Succeed())

		Expect(stored).To(Equal(config))
	})

	It("returns ErrIdentifierTaken and leaves no rows when identifier is taken", func(ctx SpecContext) {
		tx := testTx(ctx)
		config := PasswordConfig{HashedPassword: testHashedPassword}
		_, err := CreatePasswordIdentity(ctx, tx, Traits{Email: testEmail}, testEmail, config)
		Expect(err).NotTo(HaveOccurred())

		_, err = CreatePasswordIdentity(ctx, tx, Traits{Email: testEmail}, testEmail, config)
		Expect(err).To(MatchError(ErrIdentifierTaken))

		var count int
		Expect(tx.QueryRow(ctx, `SELECT count(*) FROM identities`).Scan(&count)).To(Succeed())
		Expect(count).To(Equal(1))
	})
})
