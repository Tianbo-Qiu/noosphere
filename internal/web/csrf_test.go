package web

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("maskToken and unmaskToken", func() {
	It("gives back the token", func() {
		Expect(unmaskToken(maskToken("47ISB3EWWBGSHHZI4RDKWEB3DG"))).To(Equal("47ISB3EWWBGSHHZI4RDKWEB3DG"))
	})

	It("masks the same token differently every time", func() {
		Expect(maskToken("47ISB3EWWBGSHHZI4RDKWEB3DG")).NotTo(Equal(maskToken("47ISB3EWWBGSHHZI4RDKWEB3DG")))
	})

	It("produces random bytes plus the token XOR them, twice the token's length before encoding", func() {
		Expect(maskToken("47ISB3EWWBGSHHZI4RDKWEB3DG")).To(HaveLen(70)) // 52 bytes in URL-safe base64 without padding
	})

	DescribeTable("rejects a value that maskToken cannot have produced",
		func(field string) {
			_, err := unmaskToken(field)

			Expect(err).To(MatchError(errMalformedCSRFToken))
		},
		Entry("empty", ""),
		Entry("not base64", "not base64!"),
		Entry("an odd number of bytes", "AAAA"), // decodes to 3 bytes, which cannot split into two halves
		Entry("unused trailing bits set", "AAB"), // two bytes, whose canonical encoding is "AAA"
	)

	It("compares tokens only when equal in full", func() {
		Expect(tokensEqual("47ISB3EWWBGSHHZI4RDKWEB3DG", "47ISB3EWWBGSHHZI4RDKWEB3DG")).To(BeTrue())
		Expect(tokensEqual("47ISB3EWWBGSHHZI4RDKWEB3DG", "47ISB3EWWBGSHHZI4RDKWEB3DH")).To(BeFalse())
		Expect(tokensEqual("47ISB3EWWBGSHHZI4RDKWEB3DG", "47ISB3")).To(BeFalse())
	})
})
