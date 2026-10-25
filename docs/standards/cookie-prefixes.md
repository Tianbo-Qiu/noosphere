# Cookie prefixes

Cookie prefixes are special starts of a cookie's name, `__Host-` and `__Secure-`, that make the browser enforce rules about how the cookie was set. A cookie named `__Host-csrf_token` is guaranteed to have been set over HTTPS by the exact host that receives it, and by no other.

## What problem it solves

A server cannot tell from a cookie who set it. Normally a site on one subdomain, say `blog.example.com`, may set a cookie with `Domain=example.com`, and the browser then sends that cookie to `auth.example.com` too. If an attacker controls any subdomain, through a vulnerable app or a forgotten DNS record, they can plant or overwrite cookies for the whole site. This is called **cookie tossing**. It breaks defenses that trust a cookie's value, such as a CSRF token kept in a cookie or a session cookie the attacker swaps for one of their own.

## Where it is defined

- The prefixes are specified in the IETF's revision of the cookie standard, [RFC 6265bis](https://datatracker.ietf.org/doc/draft-ietf-httpbis-rfc6265bis/) ("Cookies: HTTP State Management Mechanism").
- [MDN's Set-Cookie reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Set-Cookie#cookie_prefixes) summarizes the rules and browser behavior.
- The [OWASP CSRF cheat sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html) recommends the `__Host-` prefix for CSRF cookies, since such a cookie "cannot be (over)written from another subdomain".

## Example

Which `Set-Cookie` headers a browser accepts, from an HTTPS page (examples from MDN):

| `Set-Cookie` header | Result | Why |
|---|---|---|
| `__Host-ID=123; Secure; Path=/` | accepted | meets every `__Host-` rule |
| `__Host-id=1; Secure` | rejected | missing `Path=/` |
| `__Host-id=1; Secure; Path=/; Domain=example.com` | rejected | has a `Domain` |
| `__Secure-ID=123; Secure; Domain=example.com` | accepted | `__Secure-` only requires `Secure` |
| `__Secure-id=1` | rejected | missing `Secure` |

noosphere's CSRF cookie over HTTPS:

```
Set-Cookie: __Host-csrf_token=47ISB3EWWBGSHHZI4RDKWEB3DG; Path=/; HttpOnly; Secure; SameSite=Lax
```

## How it works

When a browser receives a cookie whose name starts with a prefix, it checks the cookie's attributes and drops the cookie if any rule is broken:

| Prefix | Requires | Forbids |
|---|---|---|
| `__Secure-` | `Secure`, set from an HTTPS page | nothing more |
| `__Host-` | `Secure`, set from an HTTPS page, `Path=/` | a `Domain` attribute |

Forbidding `Domain` is what makes `__Host-` strong. A cookie without a `Domain` attribute is **host-only**: the browser returns it "only to the host that sent it", not to subdomains, and only that host can set it. So `blog.example.com` cannot set a `__Host-` cookie that `auth.example.com` would receive. Requiring `Path=/` means the cookie applies to the whole host and cannot be shadowed by another cookie of the same name on a narrower path.

Two newer prefixes, `__Http-` and `__Host-Http-`, also require `HttpOnly`, proving the cookie was set by a server response and not by JavaScript. Browsers that do not support prefixes at all accept prefixed cookies like any other, so the prefix adds protection without breaking anything.
