# Session management

Session management is how a website remembers, from one request to the next, that a browser has signed in. After sign-in, the server gives the browser a random **session token**, usually in a cookie. The browser sends it back with every request, and the server looks it up to find out who is asking.

## What problem it solves

HTTP is stateless: each request arrives on its own, and nothing in it says who sent it. Without sessions, a user would have to send their password with every request. With sessions done badly, the token becomes the weak point, because whoever presents it is treated as the user:

- A short or predictable token can be guessed, and the attacker is signed in as someone else.
- A token that page scripts can read is stolen by any [cross-site scripting](https://owasp.org/www-community/attacks/xss/) bug.
- A token chosen before sign-in, for example one an attacker planted in the victim's browser, still works after the victim signs in. This is called **session fixation**.
- A token that never expires, or still works after logout, stays useful to whoever copied it.
- A server that stores tokens as they are hands every signed-in account to anyone who reads its database or a backup.

## Where it is defined

- [NIST SP 800-63B-4](https://pages.nist.gov/800-63-4/sp800-63b.html), Section 5 ("Session Management"), is the normative rule set. Sec. 5.1 covers the session secret, Sec. 5.1.1 the cookie that carries it, and Sec. 5.2 how long a session may last before the user signs in again. Sec. 2.1.3 sets that limit for password sign-in. See [NIST SP 800-63](nist-sp-800-63.md).
- The [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html) is practical guidance for web applications: token entropy, cookie attributes, storage, renewal at sign-in, expiry and caching. See [OWASP](owasp.md).
- Cookies themselves are defined in [RFC 6265](https://www.rfc-editor.org/rfc/rfc6265), and its revision [RFC 6265bis](https://datatracker.ietf.org/doc/draft-ietf-httpbis-rfc6265bis/) adds `SameSite` and the `__Host-` prefix ([Cookie prefixes](cookie-prefixes.md)).

## Example

A browser signs up. The server creates a random token and answers with this cookie:

```
Set-Cookie: __Host-session=5SV5CCQ26THIGSRJIVF7O244LG; Path=/; Expires=Tue, 27 Oct 2026 09:30:00 GMT; HttpOnly; Secure; SameSite=Lax
```

It stores a session record with the token's SHA-256 hash, not the token:

| token_hash | active | expires_at |
|---|---|---|
| `\x85f17d8d70e82403065f3a9615a128d606e790b71def53326da470272f859bb6` | `true` | 2026-10-27 09:30 UTC |

On each later request the browser sends `Cookie: __Host-session=5SV5CCQ26THIGSRJIVF7O244LG`. The server hashes the value, finds the record, and checks it:

| The request carries | Result |
|---|---|
| the token, before `expires_at`, record active | signed in |
| no cookie, or a value with no matching record | anonymous |
| the token, after `expires_at` | anonymous; the user signs in again |
| the token, after logout set the record inactive | anonymous |

Someone who reads the table sees only the hash. It cannot be turned back into the token, and the server accepts only the token.

## How it works

Each requirement closes one of the problems above:

| Requirement | Source |
|---|---|
| The token comes from a cryptographically secure random generator, with at least 64 bits | NIST Sec. 5.1 |
| At least 128 bits, preferably 160 or more | OWASP |
| The token is created by the server in direct response to sign-in, never taken from the browser | NIST Sec. 5.1; OWASP, renewal after a privilege change |
| The cookie is `Secure`, has the `__Host-` prefix with `Path=/`, is `HttpOnly`, has `SameSite=Lax` or `Strict`, expires with the session, and holds only an opaque value | NIST Sec. 5.1.1 |
| The server stores a one-way hash of the token; a fast hash is enough | OWASP, server-side session token storage |
| The session has an overall time limit after which the user signs in again: at most 30 days for password sign-in | NIST Sec. 2.1.3 and Sec. 5.2 |
| Logout invalidates the session on the server, not only in the browser | NIST Sec. 5.1 |
| Form posts carry a CSRF token the server checks | NIST Sec. 5.1 |
| Pages that show a signed-in user's data are sent with `Cache-Control: no-store` | OWASP, web content caching |

A fast hash such as SHA-256 is enough, unlike for [passwords](password-hashing.md), because a random token of 128 bits cannot be guessed however fast each guess is. The cookie's `Expires` only keeps browsers from piling up old cookies; NIST says it "SHALL NOT be relied upon to enforce session timeouts", so the server checks `expires_at` itself. An **inactivity timeout**, which ends a session nobody has used for a while, is optional for password sign-in and required at higher assurance levels.

In Go, [`crypto/rand.Text`](https://pkg.go.dev/crypto/rand#Text) returns a 128-bit random token, and [`crypto/sha256`](https://pkg.go.dev/crypto/sha256) hashes it.
