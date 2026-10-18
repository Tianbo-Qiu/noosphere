package identifier

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Normalize", func() {
	DescribeTable(
		"maps equivalent spellings to one form",
		func(input, want string) {
			got, err := Normalize(input)

			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(want))
		},
		Entry("keeps an already normalized email", "test@example.com", "test@example.com"),
		Entry("lowercases", "Test@Example.com", "test@example.com"),
		Entry("lowercases non-ASCII letters", "CAF\u00c9@example.com", "caf\u00e9@example.com"),                      // uppercase É → lowercase é
		Entry("maps fullwidth letters", "ｔｅｓｔ@example.com", "test@example.com"),                                      // fullwidth "test"
		Entry("composes a letter and its combining accent (NFC)", "cafe\u0301@example.com", "caf\u00e9@example.com"), // e + combining acute (2 code points) → é (1 code point)
		Entry("trims a leading space", " test@example.com", "test@example.com"),
		Entry("trims trailing whitespace", "test@example.com\t\n", "test@example.com"),
	)

	DescribeTable(
		"rejects disallowed characters",
		func(input string) {
			_, err := Normalize(input)

			Expect(err).To(HaveOccurred())
		},
		Entry("zero-width space", "test\u200b@example.com"),       // zero-width space after "test"
		Entry("right-to-left override", "test\u202e@example.com"), // reverses the display order of what follows
		Entry("no-break space inside", "test\u00a0@example.com"),  // looks like a normal space
		Entry("space inside", "test @example.com"),
	)

	DescribeTable(
		"rejects an empty identifier",
		func(input string) {
			_, err := Normalize(input)

			Expect(err).To(MatchError(ErrEmpty))
		},
		Entry("empty string", ""),
		Entry("whitespace only", " \t\n"),
	)
})
