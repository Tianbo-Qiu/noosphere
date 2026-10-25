# Fetch Metadata

Fetch Metadata is a set of request headers that browsers add to tell the server where a request came from and why it was made. The most useful one, `Sec-Fetch-Site`, says whether a request was sent by the same site, a related site, an unrelated site, or directly by the user.

## What problem it solves

In a cross-site request forgery (CSRF) attack, a page on another site makes the user's browser send a request to your site, for example by submitting a hidden form. The browser attaches the user's cookies, so the request looks like the user's own. Before Fetch Metadata, a server could not reliably tell such a request from a genuine one, and had to rely on secret tokens in every form. With `Sec-Fetch-Site`, the browser itself reports that the form was submitted from another site, and the server can refuse it outright.

## Where it is defined

- The W3C's [Fetch Metadata Request Headers](https://w3c.github.io/webappsec-fetch-metadata/#sec-fetch-site-header) specification defines the headers.
- [MDN](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Sec-Fetch-Site) lists `Sec-Fetch-Site` as widely available, "across browsers since March 2023".
- The [OWASP CSRF cheat sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html) calls `Sec-Fetch-Site` "the most useful Fetch Metadata header for blocking CSRF-like cross-origin requests", and requires checking the `Origin` header as a fallback for older browsers.

## Example

A sign-up form at `https://auth.example.com/registration`, submitted in different ways:

| How the form was submitted | `Sec-Fetch-Site` | Should the server accept it? |
|---|---|---|
| The user fills in the form on `auth.example.com` | `same-origin` | yes |
| The user opens a bookmarked URL | `none` | yes |
| A page on `blog.example.com` posts to it | `same-site` | no: another origin, even within the same site |
| A page on `attacker.example` posts to it | `cross-site` | no |

## How it works

The browser computes the value by comparing the page that started the request with the server receiving it:

| Value | Meaning |
|---|---|
| `same-origin` | same scheme, host and port |
| `same-site` | same site (same registrable domain and scheme), but a different origin, such as another subdomain |
| `cross-site` | an unrelated site |
| `none` | started by the user directly: typing a URL, opening a bookmark |

Pages cannot fake it: `Sec-Fetch-Site` is a forbidden request header, so neither JavaScript nor a form can set or change it. Browsers send it only to trustworthy URLs, meaning HTTPS or localhost.

A server applies a simple policy: allow safe methods (`GET`, `HEAD`, `OPTIONS`) always, which is why those must never change data; allow other methods only when the value is `same-origin` or `none`; reject the rest with `403`. When a request has no `Sec-Fetch-Site`, the server compares the `Origin` header's host with its own; when it has neither header, it comes from something other than a browser, such as `curl`, which cannot carry a victim's cookies.

Fetch Metadata does not replace CSRF tokens; it adds a layer that works even if a token check has a bug. In Go, [`http.CrossOriginProtection`](https://pkg.go.dev/net/http#CrossOriginProtection) (Go 1.25) implements exactly this policy.
