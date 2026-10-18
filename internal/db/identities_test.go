package db

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateIdentity", func() {
	It("stores the traits and generates a UUID v7", func(ctx SpecContext) {
		q := New(testTx(ctx))
		traits := Traits{Email: "test@example.com", FirstName: "Test", LastName: "User"}

		got, err := q.CreateIdentity(ctx, traits)

		Expect(err).NotTo(HaveOccurred())
		Expect(got.ID.Version()).To(BeEquivalentTo(7))
		Expect(got.Traits).To(Equal(traits))
		Expect(got.CreatedAt).NotTo(BeZero())
		Expect(got.UpdatedAt).NotTo(BeZero())
	})
})
