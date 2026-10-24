package password

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/text/secure/precis"
)

// Argon2id parameters: the minimum configuration recommended by the OWASP
// Password Storage Cheat Sheet, with a 16-byte salt and 32-byte key.
const (
	memoryKiB   = 19 * 1024
	iterations  = 2
	parallelism = 1
	saltLength  = 16
	keyLength   = 32
)

// Length limits, in characters. Following NIST SP 800-63B-4, each Unicode
// code point of the normalized password counts as one character.
const (
	MinLength = 8
	MaxLength = 64
)

var (
	ErrTooShort         = errors.New("password is too short")
	ErrTooLong          = errors.New("password is too long")
	ErrInvalidCharacter = errors.New("password contains a character that cannot be used")
)

// Validate checks that the password can be used: it must normalize with the
// PRECIS OpaqueString profile, and the result must be between MinLength and
// MaxLength characters long.
func Validate(password string) error {
	if password == "" {
		return ErrTooShort
	}
	normalized, err := precis.OpaqueString.String(password)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidCharacter, err)
	}
	n := utf8.RuneCountInString(normalized)
	if n < MinLength {
		return ErrTooShort
	}
	if n > MaxLength {
		return ErrTooLong
	}
	return nil
}

// Hash normalizes the password with the PRECIS OpaqueString profile and
// returns its Argon2id hash in PHC string format, for example
// $argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>.
func Hash(password string) (string, error) {
	normalized, err := precis.OpaqueString.String(password)
	if err != nil {
		return "", fmt.Errorf("normalize password: %w", err)
	}

	salt := make([]byte, saltLength)
	rand.Read(salt)
	key := argon2.IDKey([]byte(normalized), salt, iterations, memoryKiB, parallelism, keyLength)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memoryKiB, iterations, parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}
