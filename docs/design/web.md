# Web

Noosphere renders its own pages on the server, in Go. The goal is a small, readable web layer that looks good enough and is easy to test, not a full product UI.

## Server-rendered pages

The browser asks for a page, and the server returns finished HTML with a form. The form posts back to the same server, which either redirects or renders the page again with errors. Identity providers commonly work this way: their login pages live on their own domain and are rendered by the server, and apps send users there and get them back through redirects.

This keeps the pages that handle passwords simple. Every page works without JavaScript, and nothing is loaded from other domains. A small first-party script adds conveniences on top (see [Password field](#password-field)), an approach called [progressive enhancement](https://developer.mozilla.org/en-US/docs/Glossary/Progressive_Enhancement).

The alternative was a JSON API with a separate single-page app, for example in React. That is how many product apps are built, but it means a second codebase, a build step, and cross-origin setup, which this project does not need yet.

## Flows as data

Each request handler first builds the flow as plain data, such as its ID, CSRF token, fields and errors, and only then renders it as HTML. Tests can check that data directly, and a JSON response can later be added from the same data for clients that are not browsers, such as a mobile app.

## Templates and styling

- **Templates** use [`html/template`](https://pkg.go.dev/html/template) from the standard library. It escapes every value according to where it appears (HTML text, attribute, URL or script), which prevents cross-site scripting by default. [templ](https://templ.guide/) was the alternative: type-checked, but it adds a code generator.
- **Files** (templates and CSS) are compiled into the binary with [`embed`](https://pkg.go.dev/embed), so tests load exactly what production serves.
- **Styling** comes from [Pico CSS](https://picocss.com/) 2.1.1, in its classless build: plain HTML elements look clean without class names, and invalid fields can be marked with `aria-invalid`. It is copied into the repository unchanged and served by noosphere itself rather than a CDN, so the password page loads no outside code. A small `app.css` after it keeps the text at normal size and the page in a narrow column, which suits forms.

## Routing and middleware

Routes are registered on a [chi](https://github.com/go-chi/chi) router, which keeps handlers as plain `net/http` functions and lists middleware in order:

```go
r := chi.NewRouter()
r.Use(withRequestID)
r.Use(s.withAccessLog)
r.Handle("/static/*", http.FileServerFS(staticFS))
r.Get("/registration", s.getRegistration)
```

`withRequestID` gives each request an ID for logging, and `withAccessLog` writes one log line per request with its status and duration (see [Overview](overview.md#logging)). Unknown paths get `404` and wrong methods `405` from the router. A page URL with a trailing slash, such as `/registration/`, is redirected to the URL without it ([`middleware.RedirectSlashes`](https://pkg.go.dev/github.com/go-chi/chi/v5/middleware#RedirectSlashes)), so typed or autocompleted addresses still work.

## Form fields

The registration form asks for as little as it needs, and says plainly what is optional.

| Field | Required | Stored as |
|---|---|---|
| Email | yes | The profile keeps it as typed, without surrounding spaces; the login identifier is normalized with [PRECIS](../standards/precis.md) |
| First name | no | As typed, without surrounding spaces |
| Last name | no | As typed, without surrounding spaces |
| Password | yes, 8 to 64 characters | Only as an [Argon2id hash](../standards/password-hashing.md) |

The names are optional because many people do not fit a first-and-last-name pattern. The W3C's guidance on [personal names around the world](https://www.w3.org/International/questions/qa-personal-names) says "don't require that people supply a family name": many people in parts of Southern India, Malaysia and Indonesia have only a given name, and Icelanders have no family name. A required field would make them type something false.

Optional fields say so in their label, as in "First name (optional)", and required fields have plain labels. This follows the [GOV.UK Design System](https://design-system.service.gov.uk/patterns/question-pages/): "add '(optional)' to the labels of optional fields" and "never mark mandatory fields with asterisks". The word is part of the label, so screen readers read it out with the field, and nobody has to look up what a symbol means.

A password must be 8 to 64 characters long, counted after normalization with each Unicode code point as one character, so an emoji or an accented letter counts once however it was typed. These limits follow [NIST SP 800-63B-4](https://pages.nist.gov/800-63-4/sp800-63b.html) (section 3.1.1.2): 8 is its minimum for a password used together with a second factor, and verifiers should allow at least 64. Used as the only factor, NIST requires 15, so the minimum is expected to rise, or a second factor to arrive. There are no rules about mixing character types, which NIST forbids. The form states the rule under the field ("Use 8 to 64 characters."), and a password outside it gets its own message.

Each field carries the standard [`autocomplete`](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Attributes/autocomplete) value (`email`, `given-name`, `family-name`, `new-password`), so browsers and password managers can fill the form and offer to generate a strong password. When the form is shown again after an error, the typed values come back, but the password field is always empty: the password is never stored in the flow or sent back in a page.

Surrounding spaces are removed on the server. Browsers already strip them from `type="email"` fields before submitting, but scripts and other clients can send anything, so the server does not rely on the browser.

## Password field

The password field has an eye icon at its right end, so people can check what they typed before submitting. Its behavior follows the [GOV.UK password input](https://design-system.service.gov.uk/components/password-input/) component:

- The password is hidden by default. The icon is a button: an open eye shows the password, a crossed-out eye hides it again. Because it has no visible text, its accessible name ("Show password" or "Hide password") tells screen reader users what it does, and `aria-controls` points at the field. The icons come from [Feather](https://feathericons.com) and are drawn inline, so nothing is loaded from elsewhere.
- The password is hidden again when the form is submitted, so the browser does not keep it as plain text form data.
- The field has `spellcheck="false"` and `autocapitalize="none"`, so the browser neither underlines nor changes what is typed once it is visible.

Switching a field between hidden and visible needs JavaScript, so the button is added by `static/password-toggle.js`, served by noosphere itself. The HTML only marks the field with `data-password-toggle`; without JavaScript it stays an ordinary password field. The script is a file rather than inline code, so a strict content security policy that allows only the site's own scripts remains possible.

## Password strength meter

When `PASSWORD_STRENGTH_METER=true`, a meter under the password field shows how hard the typed password would be to guess, with a short explanation. It is off by default. It is advice only: the server enforces the password rules (length, the context check and the breached password check), not a score. [NIST SP 800-63B-4](../standards/nist-sp-800-63.md) asks for guidance that helps people choose a strong password, and a meter is one form of it.

The estimate comes from [zxcvbn-ts](https://zxcvbn-ts.github.io/zxcvbn/) (MIT license), the maintained TypeScript version of Dropbox's zxcvbn. Instead of counting character types, it estimates how many guesses an attacker would need: it splits the password into pieces it recognizes, such as common passwords, dictionary words, names, keyboard runs, dates and repeats, and adds up the cost of guessing each piece. It turns the estimate into a score from 0 to 4, shown with the words from its documentation:

| Score | Shown as | Estimated guesses |
|---|---|---|
| 0 | Too guessable | fewer than 10^3 |
| 1 | Very guessable | fewer than 10^6 |
| 2 | Somewhat guessable | fewer than 10^8 |
| 3 | Safely unguessable | fewer than 10^10 |
| 4 | Very unguessable | 10^10 or more |

The values of the other fields, such as the email and the names, are passed to zxcvbn as user inputs, so a password made of them scores low, like in the context check. With `test@example.com` in the email field, the meter shows:

| Password | Score | Text |
|---|---|---|
| `password` | 0 | Too guessable. This is a heavily used password. Add more words that are less common. |
| `test@example.com` | 0 | Too guessable. There should not be any personal or page related data. Add more words that are less common. |
| `margaretgarden` | 2 | Somewhat guessable. This is similar to a commonly used password. Add more words that are less common. |
| `kq7#Vz9!pL2m` | 4 | Very unguessable. |

The bar is an HTML `<meter>` with `low="2"`, `high="3"` and `optimum="4"`, so browsers draw it red for scores 0 and 1, yellow for 2 and green for 3 and 4; at score 0 the bar is empty and only the text is shown. The text is a polite live region, so screen readers announce it; the bar repeats it and is hidden from them.

zxcvbn needs word lists to recognize words, and they are large. noosphere uses zxcvbn-ts's ready-made browser builds, copied into `static/zxcvbn-ts/` unchanged, with no build step:

| File | Contains | Uncompressed | Gzipped |
|---|---|---|---|
| `core.js` | the estimator | 107,643 bytes | 25,094 bytes |
| `language-common.js` | common passwords, keyboard layouts, diceware words | 429,162 bytes | 226,446 bytes |
| `language-en.js` | English words, first names, last names | 1,201,803 bytes | 620,975 bytes |

The English lists are included, so dictionary words and names count as easy to guess. Each build is a classic script that sets `window.zxcvbnts`, so the page loads them with `<script defer>` tags followed by `static/password-strength.js`, which adds the meter. Deferred scripts download while the page is parsed, never block it from showing, and run in order once it has been parsed, so the meter is usually ready before the user reaches the password field. When the flag is off, the page contains none of these tags, and nothing is downloaded. As with the eye icon, the field is only marked with `data-password-strength`, and without JavaScript nothing changes.

## Context check

Sign-up refuses a password that is a word from the sign-up itself, with the message "Don't use your email, your name or the site's name as your password." The words are the email, the part of the email before the `@`, the first and last name if given, and the site's name, `noosphere`.

The whole password is compared to each word, ignoring case and after the same normalization used for hashing, as [NIST SP 800-63B-4](../standards/nist-sp-800-63.md) requires: `Test@Example.com` is refused for the account `test@example.com`, but `test@example.com and a lot more` is accepted. Passwords that are only similar to the email are not refused, so the reason for a refusal is always visible to the user ([Context-specific passwords](../standards/context-specific-passwords.md)).

It is on by default (`PASSWORD_CONTEXT_CHECK=true`); `make run` turns it off, like the breached password check, so quick local sign-ups are not refused. The check costs nothing and needs no network, so it runs before the breached password check, and a password it refuses is never sent to the outside service.

## Breached password check

Sign-up refuses a password that has appeared in a data breach, with the message "This password has appeared in a data breach. Choose a different one." The check asks the [Pwned Passwords](../standards/breached-password-check.md) range API, sending only the first 5 characters of the password's SHA-1 hash and asking for a padded response, so the service learns neither the password nor which one was checked.

It is on by default (`PWNED_PASSWORDS_CHECK=true`). Local development turns it off in `make run`, so sign-up works offline and test sign-ups send no requests to an outside service. When it is on:

- It runs only for a password that already passed the length and character rules, so invalid input never causes a request.
- It **fails open**: if the service answers with an error or takes longer than `PWNED_PASSWORDS_TIMEOUT` (2 seconds by default), the password is accepted and a warning is logged. Sign-up never depends on a service outside noosphere's control.
- `PWNED_PASSWORDS_URL` can point at a self-hosted copy of the API instead of the public one.

Tests run against a small fake of the range API, so they do not depend on the internet.

## Errors

Errors fall into two kinds, and each is always handled the same way:

| Kind | Examples | Response |
|---|---|---|
| The user can fix it | email already registered, invalid email, empty password | The form again, with what the user typed (never the password) and a message under each invalid field |
| The form cannot continue | CSRF check failed, cookies blocked (`403`), database unavailable (`500`) | An error page in the site layout, with a link to start again |

A form error is saved in the flow's `state` (the typed values and one message per field), and the browser is redirected back to the form, which renders from the flow. So every page a user sees comes from a `GET`, and refreshing it never resubmits the form. The invalid field is marked with `aria-invalid="true"` and linked to its message with `aria-describedby`, which screen readers announce and Pico styles in red. Checks that cost nothing run first and report together, so a user sees every simple mistake at once; the slow password hash and the database's uniqueness check run only after those pass.

The details of a `500` are only logged, never shown. Plain-text `http.Error` is used in one place: when rendering a page fails, since rendering is then what broke.

## Cross-origin protection

Every form post, and any other request that could change something, is first checked for where it came from. Browsers label each request with the `Sec-Fetch-Site` header ([Fetch Metadata](../standards/fetch-metadata.md)) (`same-origin`, `same-site`, `cross-site` or `none`), and older ones with [`Origin`](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Origin). A post that a browser marks as coming from another site is rejected with the "Request rejected" page and a `403`, before the form is even read.

This uses Go's own [`http.CrossOriginProtection`](https://pkg.go.dev/net/http#CrossOriginProtection) (Go 1.25), which checks `Sec-Fetch-Site`, "available in all browsers since 2023", and falls back to comparing the `Origin` header's host with the request's. `GET`, `HEAD` and `OPTIONS` are always allowed, which is why no `GET` handler may change state. A request with neither header comes from something other than a browser, such as `curl`, and is allowed; such a client cannot carry a victim's cookies, so it is not a CSRF risk.

It is a second, independent layer in front of the CSRF token: the [OWASP CSRF cheat sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html) calls `Sec-Fetch-Site` "the most useful Fetch Metadata header for blocking CSRF-like cross-origin requests" and requires an `Origin` check as the fallback, which is exactly what the Go middleware does. Rejections are logged as warnings with the request's `Sec-Fetch-Site` and `Origin` values.

## CSRF cookie

Each browser gets one random CSRF token, in an `HttpOnly`, `SameSite=Lax` cookie that is `Secure` outside local development. Over HTTPS the cookie is named `__Host-csrf_token`: browsers accept the `__Host-` prefix ([Cookie prefixes](../standards/cookie-prefixes.md)) only for a `Secure` cookie with `Path=/` and no `Domain`, so no other subdomain of the site can set or overwrite it, which closes the main weakness [OWASP](../standards/owasp.md) notes for this pattern. Over plain HTTP in local development, where browsers would refuse that prefix, it is named `csrf_token`. Every flow the browser starts stores a copy, and a form post must carry the same token as both the cookie and the flow. One token per browser, rather than per flow, lets a user have sign-up open in two tabs.

A browser that does not send the cookie back would otherwise get a new flow, and a new cookie, on every request. Instead it gets a `403` page asking it to allow cookies.

## Session cookie

A signed-in browser is recognized by its session cookie. The cookie holds a random token; the server keeps a session record for it, and a request counts as signed in only while that record exists, is active and has not expired. The rules it follows are on [Session management](../standards/session-management.md).

The token is made with [`crypto/rand.Text`](https://pkg.go.dev/crypto/rand#Text), 128 random bits written as 26 base32 characters, like the CSRF token. That meets NIST's minimum of 64 bits and OWASP's of 128, but not the 160 bits or more that [OWASP](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html) prefers; the standard library function is used as it is rather than a hand-written generator, and 2^128 possible tokens are already far beyond guessing. The `sessions` table stores only its SHA-256 hash, so a leaked database or backup holds no token that could be put in a cookie. A fast hash is enough because a random 128-bit token cannot be guessed; passwords need a slow one because people choose them.

The cookie is `HttpOnly`, `SameSite=Lax` and `Path=/`, and expires with the session. Like the CSRF cookie, it is `Secure` and named `__Host-session` over HTTPS, so no other subdomain can set or overwrite it, and named `session` over plain HTTP in local development. For example:

```
Set-Cookie: __Host-session=5SV5CCQ26THIGSRJIVF7O244LG; Path=/; Expires=Tue, 27 Oct 2026 09:30:00 GMT; HttpOnly; Secure; SameSite=Lax
```

A session lasts 24 hours from sign-in (`SessionLifetime`), after which the user signs in again. NIST allows up to 30 days for password sign-in. The cookie's expiry only keeps the browser from holding a dead cookie; the server checks the session's own `expires_at`. The session's ID, never its token, is what appears in logs.

## Signed in after sign-up

A successful sign-up also signs the user in, so the account can be used right away without typing the password again.

The session is created in the same database transaction as the identity, its password credential and its identifier. If any of the four inserts fails, none is kept. Creating the session after the account was committed would risk an account with no session after a failure, and since signing up again with the same email is refused, the user would be stuck until they can log in.

The token is created on the server at that moment, never taken from the browser, so a token planted in the browser before sign-up cannot become a signed-in session ([session fixation](https://owasp.org/www-community/attacks/Session_fixation)). The response that redirects after sign-up sets the session cookie, and the log records `signed up` with the identity and session IDs.

## Compression

noosphere does not compress its responses. Compression is left to whatever runs in front of it in a deployment, such as a reverse proxy or a CDN.

**What compression does.** The server compresses a response before sending it, and the browser decompresses it. Text shrinks a lot because it repeats itself; with gzip at level 5, noosphere's files measured:

| File | Uncompressed | Gzipped |
|---|---|---|
| `pico.classless.min.css` | 71,040 bytes | 10,684 bytes |
| `app.css` | 1,882 bytes | 767 bytes |
| `password-toggle.js` | 1,729 bytes | 891 bytes |
| `password-strength.js` | 2,360 bytes | 1,129 bytes |
| `zxcvbn-ts/` (three files) | 1,738,608 bytes | 872,515 bytes |

The browser lists what it can decode in `Accept-Encoding`, the server names what it used in `Content-Encoding`, and `Vary: Accept-Encoding` keeps caches from mixing the two versions.

**Why not in the application.** The usual deployment already has a reverse proxy or CDN in front of the server, and those compress with less effort and more formats (gzip, Brotli, zstd) than the app would. On a developer's machine, size does not matter. Doing it in the app as well would add code and still leave the proxy's settings to decide what is actually sent.

**What has to be safe instead.** Proxies compress HTML by default: [nginx](https://nginx.org/en/docs/http/ngx_http_gzip_module.html) says "responses with the 'text/html' type are always compressed", and [Caddy](https://caddyserver.com/docs/caddyfile/directives/encode) compresses all `text/*` responses. Compressed HTML is open to [BREACH](https://en.wikipedia.org/wiki/BREACH) when a page contains a secret and text an attacker can influence, because guesses that match the secret make the response smaller. The sign-up page is such a page: it carries the CSRF token and shows the typed email again after an error. Rather than depend on every deployment's proxy settings, the page itself is made safe: the CSRF token in it is masked with fresh random bytes on every response ([Masked CSRF token](#masked-csrf-token)), so compression has nothing stable to reveal. noosphere can then sit behind any proxy, with compression on or off.

## Masked CSRF token

The hidden `csrf_token` field in a form never contains the token itself. Each time the page is rendered, the server picks random bytes as long as the token, XORs them with the token, and puts the random bytes followed by the result in the field, encoded as URL-safe base64. For the token `47ISB3EWWBGSHHZI4RDKWEB3DG`, two renders of the same form produced:

```
LJdYMr9EywtT_VLfyYxC5m6nUldJOOyfuY0YoBFh_XeOXAS_FYyBxBivWvUWHB59rqz9yg
G5eSYzW34b_CHrQQx3PzM-JqCW4yocECIqUvoNswd4Sk6JVc80OPO6l61jhNJWXkgzFm4g
```

On a form post, the server splits the field into its two halves and XORs them again, which gives back the token, then compares it with the cookie and the flow's copy. Anything that does not decode, does not split into two equal halves, or is not in canonical base64 is rejected like a wrong token.

This defends against [BREACH](https://en.wikipedia.org/wiki/BREACH), an attack on compressed HTTPS responses. A page that contains both a secret and text an attacker can influence, such as the email shown again after an error, leaks the secret through its compressed size: guesses that match part of the secret compress better. noosphere does not compress pages itself, but a reverse proxy or CDN in front of it usually does, since [nginx](https://nginx.org/en/docs/http/ngx_http_gzip_module.html) and [Caddy](https://caddyserver.com/docs/caddyfile/directives/encode) both compress HTML by default. With masking, the bytes in the page change on every response, so there is nothing stable for the attack to recover, wherever compression happens.

All token comparisons use [`crypto/subtle.ConstantTimeCompare`](https://pkg.go.dev/crypto/subtle#ConstantTimeCompare), which takes the same time however much of a guess is right, so response times reveal nothing about the token.

## Layout

```
internal/web/
  server.go              Config, Server and the routes
  middleware.go          request ID and access log
  csrf.go                the per-browser CSRF cookie
  session.go             the session cookie's name
  registration.go        handlers for the registration flow
  render.go              loads the embedded templates and static files, renders a page
  templates/
    layout.html          shared page frame and stylesheet links
    registration.html    the registration form
    error.html           an error message with a link to start again
  static/
    pico.classless.min.css   Pico CSS, unchanged
    app.css                  noosphere's adjustments
    password-toggle.js       the show/hide eye icon for password fields
    password-strength.js     the password strength meter
    zxcvbn-ts/               zxcvbn-ts browser builds and license, unchanged
```

`internal/web` turns requests into flow data and flow data into pages. Storage stays in `internal/db`.

## Testing

Handler tests start the whole server with [`httptest`](https://pkg.go.dev/net/http/httptest), routes and middleware included, on the real connection pool and the test database, whose tables are emptied before each test. A client with a cookie jar acts as the browser and does not follow redirects, so each response can be checked: the status code, the redirect target and the flow behind it, cookies, and the important parts of the page. No browser is needed.
