# OWASP

OWASP, the Open Worldwide Application Security Project, is a nonprofit foundation that publishes free, community-written guidance on building secure software. It started in 2001 and was incorporated as the OWASP Foundation in 2004 ([about OWASP](https://owasp.org/about/)).

## What problem it solves

Standards such as [NIST SP 800-63](nist-sp-800-63.md) say *what* a service must achieve, for example "store passwords so a leak does not reveal them". Developers also need to know *how*: which algorithm, which parameters, which header, which mistakes to avoid. OWASP fills that gap with short, practical guides kept up to date by security practitioners.

## Where it is defined

OWASP's work is organized as projects. The three most used are:

| Project | What it is |
|---|---|
| [Cheat Sheet Series](https://cheatsheetseries.owasp.org/) | "a concise collection of high value information on specific application security topics", one page per topic |
| [Top 10](https://owasp.org/www-project-top-ten/) | the ten most critical security risks to web applications, updated every few years |
| [ASVS](https://asvs.owasp.org/) | the Application Security Verification Standard, a checklist of security requirements to build and test against; version 5.0.0 was released on 30 May 2025 |

## Example

The cheat sheets noosphere follows:

| Cheat sheet | Used for |
|---|---|
| [Password Storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html) | Argon2id with at least 19 MiB of memory, 2 iterations and 1 degree of parallelism ([Password hashing](password-hashing.md)) |
| [Cross-Site Request Forgery Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html) | the CSRF token tied to the browser by a cookie |
| [Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html) | session tokens, stored only as hashes |
| [Authentication](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html) | password length rules and login error messages |

## How it works

OWASP is not a standards body in the formal sense: its documents are not binding, and anyone can contribute through its public GitHub repositories. Its value comes from being specific, current and widely reviewed. Where NIST and OWASP overlap, they usually agree, since OWASP's authentication guidance cites NIST; OWASP then adds the concrete settings and examples.
