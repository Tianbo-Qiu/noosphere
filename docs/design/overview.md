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
| Pages | Server-rendered with `html/template` (see [Web](web.md)) |
| Tests | [Ginkgo](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/) (see [Testing](testing.md)) |

## Layout

```
cmd/noosphere/          the server binary
internal/db/            storage: generated query code plus hand-written transactions
internal/identifier/    normalization of login identifiers such as emails
db/migrations/          schema changes, applied in order
db/queries/             the SQL that sqlc turns into Go
db/init/                one-time setup for a new database container (creates the test database)
```

Each package has one job. Code outside `internal/db` never writes SQL and never sees a Postgres error code: `internal/db` turns those into plain Go errors such as `ErrIdentifierTaken`.

## Everyday commands

The Makefile wraps the common tasks:

| Command | Does |
|---|---|
| `make db-up` | Starts Postgres and waits until it is ready |
| `make migrate-up` | Applies new migrations to the development database |
| `make generate` | Regenerates the Go query code after a change in `db/queries` or `db/migrations` |
| `make run` | Runs the server against the development database |
| `make test` | Migrates the test database, then runs all tests |
| `make check` | Formatting check, `go vet`, then tests; run before committing |
