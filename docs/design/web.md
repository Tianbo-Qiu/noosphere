# Web

Noosphere renders its own pages on the server, in Go. The goal is a small, readable web layer that looks good enough and is easy to test, not a full product UI.

## Server-rendered pages

The browser asks for a page, and the server returns finished HTML with a form. The form posts back to the same server, which either redirects or renders the page again with errors. Identity providers commonly work this way: their login pages live on their own domain and are rendered by the server, and apps send users there and get them back through redirects.

This keeps the pages that handle passwords simple. There is no JavaScript bundle, nothing loaded from other domains, and the pages work without JavaScript.

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

## Errors

Errors fall into two kinds, and each is always handled the same way:

| Kind | Examples | Response |
|---|---|---|
| The user can fix it | email already registered, invalid email, empty password | The form again, with what the user typed (never the password) and a message under each invalid field |
| The form cannot continue | CSRF check failed, cookies blocked (`403`), database unavailable (`500`) | An error page in the site layout, with a link to start again |

A form error is saved in the flow's `state` (the typed values and one message per field), and the browser is redirected back to the form, which renders from the flow. So every page a user sees comes from a `GET`, and refreshing it never resubmits the form. The invalid field is marked with `aria-invalid="true"` and linked to its message with `aria-describedby`, which screen readers announce and Pico styles in red. Checks that cost nothing run first and report together, so a user sees every simple mistake at once; the slow password hash and the database's uniqueness check run only after those pass.

The details of a `500` are only logged, never shown. Plain-text `http.Error` is used in one place: when rendering a page fails, since rendering is then what broke.

## CSRF cookie

Each browser gets one random CSRF token, in an `HttpOnly`, `SameSite=Lax` cookie that is `Secure` outside local development. Every flow the browser starts stores a copy, and a form post must carry the same token as both the cookie and the flow. One token per browser, rather than per flow, lets a user have sign-up open in two tabs.

A browser that does not send the cookie back would otherwise get a new flow, and a new cookie, on every request. Instead it gets a `403` page asking it to allow cookies.

## Layout

```
internal/web/
  server.go              Config, Server and the routes
  middleware.go          request ID and access log
  csrf.go                the per-browser CSRF cookie
  registration.go        handlers for the registration flow
  render.go              loads the embedded templates and static files, renders a page
  templates/
    layout.html          shared page frame and stylesheet links
    registration.html    the registration form
    error.html           an error message with a link to start again
  static/
    pico.classless.min.css   Pico CSS, unchanged
    app.css                  noosphere's adjustments
```

`internal/web` turns requests into flow data and flow data into pages. Storage stays in `internal/db`.

## Testing

Handler tests start the whole server with [`httptest`](https://pkg.go.dev/net/http/httptest), routes and middleware included, on the real connection pool and the test database, whose tables are emptied before each test. A client with a cookie jar acts as the browser and does not follow redirects, so each response can be checked: the status code, the redirect target and the flow behind it, cookies, and the important parts of the page. No browser is needed.
