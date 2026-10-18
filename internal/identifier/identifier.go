package identifier

import (
	"errors"
	"strings"

	"golang.org/x/text/secure/precis"
)

var ErrEmpty = errors.New("identifier is empty")

func Normalize(identifier string) (string, error) {
	out, err := precis.UsernameCaseMapped.String(strings.TrimSpace(identifier))
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", ErrEmpty
	}
	return out, nil
}
