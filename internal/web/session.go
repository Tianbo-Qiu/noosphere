package web

// sessionCookieName is the session cookie's name over plain HTTP, in local
// development. Over HTTPS it gets the __Host- prefix, like the CSRF cookie.
const sessionCookieName = "session"

// sessionCookie returns the name of the session cookie for this server.
func (s *Server) sessionCookie() string {
	if s.cfg.SecureCookies {
		return "__Host-" + sessionCookieName
	}
	return sessionCookieName
}
