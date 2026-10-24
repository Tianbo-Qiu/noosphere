# Overview

Noosphere is a single Go server backed by one Postgres database. It is built one user journey at a time, so the code only contains what the journeys so far need.

## Stack

| Part | Choice |
|---|---|
| Language | Go |
| Database | Postgres 18, run locally with Docker Compose |
| Migrations | [goose](https://github.com/pressly/goose), plain SQL files |
| Queries | [sqlc](https://sqlc.dev/), which generates typed Go functions from SQL |
| Database driver | [pgx](https://github.com/jackc/pgx) with its connection pool |
| HTTP routing | [chi](https://github.com/go-chi/chi), on top of the standard `net/http` handlers |
| Pages | Server-rendered with `html/template` (see [Web](web.md)) |
| Password hashing | Argon2id from [`golang.org/x/crypto/argon2`](https://pkg.go.dev/golang.org/x/crypto/argon2) (see [Password hashing](../standards/password-hashing.md)) |
| Logging | [`log/slog`](https://pkg.go.dev/log/slog) from the standard library |
| Tests | [Ginkgo](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/) (see [Testing](testing.md)) |
| CI | GitHub Actions: `make check` against a Postgres 18 service on every push and pull request |

## Layout

```
cmd/noosphere/          the server binary: reads settings, builds the logger, the database pool and the web server
internal/web/           HTTP: routes, middleware, handlers, templates and static files
internal/db/            storage: generated query code plus hand-written transactions
internal/identifier/    normalization of login identifiers such as emails
internal/password/      password rules and hashing
internal/pwned/         breached password check against the Pwned Passwords API
internal/logging/       builds the logger from its format and level
internal/dbtest/        test-only: a migrated database per test package
db/migrations/          schema changes, applied in order
db/queries/             the SQL that sqlc turns into Go
db/init/                one-time setup for a new database container (creates noosphere_test, the database TEST_DATABASE_URL names)
```

Each package has one job, and only `cmd/noosphere` reads environment variables; the other packages receive their settings as arguments. Code outside `internal/db` never writes SQL and never sees a Postgres error code: `internal/db` turns those into plain Go errors such as `ErrIdentifierTaken`.

## Everyday commands

The Makefile wraps the common tasks:

| Command | Does |
|---|---|
| `make db-up` | Starts Postgres and waits until it is ready |
| `make migrate-up` | Applies new migrations to the development database |
| `make generate` | Regenerates the Go query code after a change in `db/queries` or `db/migrations` |
| `make run` | Runs the server on `http://localhost:8080` against the development database |
| `make test` | Runs all tests; each test package gets its own database on the local Postgres |
| `make test-v` | The same tests through the Ginkgo runner, in random order, printing every spec; extra flags go in `ARGS` |
| `make check` | Formatting check, `go vet`, then tests; run before committing |

## Settings

The server reads its settings from environment variables at startup and stops if one is invalid.

| Variable | Default | Meaning |
|---|---|---|
| `DATABASE_URL` | none, required | Postgres connection string |
| `ADDR` | `:8080` | Address the server listens on |
| `COOKIE_SECURE` | `true` | Mark cookies `Secure` (HTTPS only); `make run` sets `false` for local HTTP |
| `LOG_FORMAT` | `text` | `text` or `json` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error` |
| `PWNED_PASSWORDS_CHECK` | `false` | Reject new passwords seen in data breaches, using the Pwned Passwords API |
| `PWNED_PASSWORDS_URL` | `https://api.pwnedpasswords.com` | The API to ask, for example a self-hosted mirror |
| `PWNED_PASSWORDS_TIMEOUT` | `2s` | How long to wait for an answer before accepting the password anyway |

## Logging

Every component gets the logger from `main` instead of using a global one. Log lines are structured: a short message plus key/value pairs, such as `flow_id`. `go vet` rejects a key without a value.

Each HTTP request gets a random `request_id`, stored in its context, returned in the `X-Request-ID` response header, and included in every line the request logs. After the handler finishes, the access log writes one `request` line with the method, path, status and duration:

```
level=INFO msg="registration flow created" request_id=B73V2TUNDMX76PUNOT6AAGWSKY flow_id=01a15cb4-e13c-700a-aad5-2de34f8a9038
level=INFO msg=request request_id=B73V2TUNDMX76PUNOT6AAGWSKY method=GET path=/registration status=303 duration=6.45475ms
```

A sign-up attempt spans several requests, so it is followed by its `flow_id`. Secrets (CSRF tokens, passwords, connection strings) and email addresses are never logged, and the access log records the path without the query string.
