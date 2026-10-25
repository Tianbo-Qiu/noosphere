package web

import (
	"crypto/rand"
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
