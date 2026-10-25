package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
)

// csrfCookieName is the CSRF cookie's name over plain HTTP, in local
// development. Over HTTPS the cookie is named with the __Host- prefix, which
// browsers only accept for a Secure cookie with Path=/ and no Domain, so no
// other subdomain can set or overwrite it.
const csrfCookieName = "csrf_token"

// csrfCookie returns the name of the CSRF cookie for this server.
func (s *Server) csrfCookie() string {
	if s.cfg.SecureCookies {
		return "__Host-" + csrfCookieName
	}
	return csrfCookieName
}

// csrfToken returns the browser's CSRF token from its cookie. If the browser
// has none yet, it creates a token and sets the cookie on the response.
func (s *Server) csrfToken(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie(s.csrfCookie()); err == nil && c.Value != "" {
		return c.Value
	}
	token := rand.Text()
	http.SetCookie(w, &http.Cookie{
		Name:     s.csrfCookie(),
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	return token
}

// errMalformedCSRFToken means a submitted form token could not be unmasked.
var errMalformedCSRFToken = errors.New("csrf token is malformed")

// maskToken returns the token in a form that differs on every call: random
// bytes R the length of the token, followed by the token XOR R, encoded as
// URL-safe base64. Pages then never repeat the same secret bytes, which
// compression attacks such as BREACH need to recover it.
func maskToken(token string) string {
	masked := make([]byte, 2*len(token))
	random, xored := masked[:len(token)], masked[len(token):]
	rand.Read(random)
	for i := range len(token) {
		xored[i] = token[i] ^ random[i]
	}
	return base64.RawURLEncoding.EncodeToString(masked)
}

// unmaskToken reverses maskToken. Decoding is strict, so a value that only
// differs from a valid one in unused trailing bits is rejected too.
func unmaskToken(field string) (string, error) {
	masked, err := base64.RawURLEncoding.Strict().DecodeString(field)
	if err != nil || len(masked) == 0 || len(masked)%2 != 0 {
		return "", errMalformedCSRFToken
	}
	n := len(masked) / 2
	token := make([]byte, n)
	for i := range n {
		token[i] = masked[i] ^ masked[n+i]
	}
	return string(token), nil
}

// tokensEqual compares two tokens in constant time, so the time taken does
// not reveal how much of a guess was right.
func tokensEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
