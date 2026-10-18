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
- **Styling** comes from one classless stylesheet, such as [Pico CSS](https://picocss.com/) or [Simple.css](https://simplecss.org/), which makes plain HTML elements look clean without class names. It is copied into the repository and served by noosphere itself rather than a CDN.

## Layout

```
internal/web/
  registration.go        handlers for the registration flow
  render.go              loads the embedded templates and renders a page
  templates/
    layout.html          shared page frame and stylesheet link
    registration.html    the registration form
  static/
    style.css            the vendored stylesheet
```

`internal/web` turns requests into flow data and flow data into pages. Storage stays in `internal/db`.

## Testing

Handler tests send real requests through [`httptest`](https://pkg.go.dev/net/http/httptest) and check the status code, redirects, cookies and the important parts of the page, such as the form, the CSRF field and error messages. No browser is needed.
