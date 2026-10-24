package password

import (
	"bytes"
	"encoding/base64"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/argon2"
)

var _ = Describe("Validate", func() {
	DescribeTable("accepts passwords of 8 to 64 characters",
		func(password string) {
			Expect(Validate(password)).To(Succeed())
		},
		Entry("8 characters", "12345678"),
		Entry("64 characters", strings.Repeat("a", 64)),
		Entry("a space counts as a character", "correct horse"),
		Entry("8 characters, one of them an emoji", "1234567\U0001F511"),             // key emoji: 4 bytes, 1 code point
		Entry("8 characters after e + combining accent becomes é", "1234567e\u0301"), // 9 code points typed, 8 after NFC
	)

	DescribeTable("rejects a password that is too short",
		func(password string) {
			Expect(Validate(password)).To(MatchError(ErrTooShort))
		},
		Entry("empty", ""),
		Entry("7 characters", "1234567"),
		Entry("7 characters after e + combining accent becomes é", "123456e\u0301"), // 8 code points typed, 7 after NFC
	)

	It("rejects a password longer than 64 characters", func() {
		Expect(Validate(strings.Repeat("a", 65))).To(MatchError(ErrTooLong))
	})

	DescribeTable("rejects a password with a character that cannot be used",
		func(password string) {
			Expect(Validate(password)).To(MatchError(ErrInvalidCharacter))
		},
		Entry("control character", "correct horse\u0007"), // BEL
		Entry("zero-width space", "correct\u200bhorse"),   // invisible
	)
})

var _ = Describe("Hash", func() {
	It("returns an Argon2id PHC string with the OWASP parameters", func() {
		hash, err := Hash("correct horse battery staple")
		Expect(err).NotTo(HaveOccurred())

		Expect(hash).To(HavePrefix("$argon2id$v=19$m=19456,t=2,p=1$"))
		salt, key := saltAndKey(hash)
		Expect(salt).To(HaveLen(16))
		Expect(key).To(HaveLen(32))
	})

	It("derives the key from the password and the stored salt", func() {
		hash, err := Hash("correct horse battery staple")
		Expect(err).NotTo(HaveOccurred())

		Expect(matches(hash, "correct horse battery staple")).To(BeTrue())
		Expect(matches(hash, "correct horse battery stapler")).To(BeFalse())
	})

	It("uses a new salt for every hash of the same password", func() {
		first, err := Hash("correct horse battery staple")
		Expect(err).NotTo(HaveOccurred())
		second, err := Hash("correct horse battery staple")
		Expect(err).NotTo(HaveOccurred())

		firstSalt, _ := saltAndKey(first)
		secondSalt, _ := saltAndKey(second)
		Expect(secondSalt).NotTo(Equal(firstSalt))
		Expect(second).NotTo(Equal(first))
	})

	It("never contains the password", func() {
		hash, err := Hash("correct horse battery staple")
		Expect(err).NotTo(HaveOccurred())

		Expect(hash).NotTo(ContainSubstring("correct horse battery staple"))
	})

	DescribeTable("hashes the normalized form of the password",
		func(typed, normalized string) {
			hash, err := Hash(typed)
			Expect(err).NotTo(HaveOccurred())

			Expect(matches(hash, normalized)).To(BeTrue())
		},
		Entry("no-break space becomes a plain space", "correct\u00a0horse", "correct horse"),    // U+00A0 looks like a space
		Entry("ideographic space becomes a plain space", "correct\u3000horse", "correct horse"), // U+3000, the full-width space in CJK input
		Entry("e + combining accent becomes é", "cafe\u0301", "caf\u00e9"),                      // 2 code points → 1 code point (NFC)
	)

	DescribeTable("keeps differences that make passwords distinct",
		func(typed, other string) {
			hash, err := Hash(typed)
			Expect(err).NotTo(HaveOccurred())

			Expect(matches(hash, other)).To(BeFalse())
		},
		Entry("uppercase and lowercase", "Secret", "secret"),
		Entry("fullwidth and plain letters", "\uff53ecret", "secret"), // fullwidth "s"
	)

	DescribeTable("rejects passwords that cannot be normalized",
		func(password string) {
			_, err := Hash(password)

			Expect(err).To(HaveOccurred())
		},
		Entry("empty", ""),
		Entry("control character", "secret\u0007"), // BEL
		Entry("zero-width space", "secret\u200b"),  // invisible
	)
})

// saltAndKey decodes the salt and the key from a PHC string.
func saltAndKey(hash string) (salt, key []byte) {
	GinkgoHelper()
	parts := strings.Split(hash, "$")
	Expect(parts).To(HaveLen(6), "PHC string %q", hash)

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	Expect(err).NotTo(HaveOccurred())
	key, err = base64.RawStdEncoding.DecodeString(parts[5])
	Expect(err).NotTo(HaveOccurred())
	return salt, key
}

// matches reports whether hashing password with the hash's salt and the
// package's parameters gives the hash's key.
func matches(hash, password string) bool {
	GinkgoHelper()
	salt, key := saltAndKey(hash)
	return bytes.Equal(argon2.IDKey([]byte(password), salt, iterations, memoryKiB, parallelism, keyLength), key)
}
