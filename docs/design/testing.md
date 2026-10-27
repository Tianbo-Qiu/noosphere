# Testing

Tests run against a real Postgres database, and each test layer proves a different thing.

## Layers

| Layer | Tests | Example |
|---|---|---|
| Unit | Pure functions, no database or network | `identifier.Normalize` maps `Test@Example.com` to `test@example.com` |
| Database | Queries, transactions and constraints against real Postgres | a second sign-up with the same email returns `ErrIdentifierTaken` |
| HTTP | Handlers through real requests with [`httptest`](https://pkg.go.dev/net/http/httptest) and a cookie jar | `POST /registration` with a valid form creates the account and redirects to the account-created page |
| End-to-end | The built server, driven like a user would | sign up, then log in |

The database is never mocked. Constraints, transactions and error codes are exactly the behavior worth testing, and a mock would only repeat what the test author already assumed.

## Isolation

Tests that share a database can see each other's rows. A test that expects one identity finds two, or a sign-up fails with "email already taken" because another test registered the same address a moment earlier. Isolation means every test starts from a known state, no matter what else is running.

There are two places where tests can meet: between test packages, and between tests inside one package.

### Between packages: a database per package

`go test ./...` builds each package into its own test binary and runs those binaries **in parallel**, as many at once as the `-p` flag allows, which defaults to the number of CPUs ([`go help build`](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies)). Two packages that use the same database therefore run at the same time.

That broke when `internal/db` and `internal/web` shared one test database. A plain `go test ./...` failed on its first run:

```
FAIL! -- 10 Passed | 12 Failed | 0 Pending | 0 Skipped
FAIL	github.com/Tianbo-Qiu/noosphere/internal/db	0.251s
```

The web tests had committed `test@example.com`, so a database test inserting the same address hit the unique constraint, and a database test counting identities saw the web tests' rows. The next 40 runs passed: a race like this fails only when the timing lines up, which is what makes it dangerous.

The options:

| Option | How | Why not, or why |
|---|---|---|
| Run packages one at a time | `go test -p 1 ./...` | `-p` also limits compiling, so builds lose their parallelism; a proposal to separate the two is still open ([golang/go#61318](https://github.com/golang/go/issues/61318)). It only protects runs that pass the flag, not an editor or a plain `go test`. |
| A shared database with a lock | Each package takes a Postgres advisory lock for its whole run | Works, but packages still wait for each other, and every package must remember to take the lock |
| **A database per package** | Each package creates, migrates and drops its own database on the same server | **Chosen.** Nothing is shared, so nothing needs coordinating |
| A database per test | Every test gets a fresh database, usually copied from a migrated template | The strongest isolation, needed only when tests inside one package run in parallel ([Michael Stapelberg](https://michael.stapelberg.ch/posts/2024-11-19-testing-with-go-and-postgresql-ephemeral-dbs/) describes this setup) |

A database per package is the usual answer to this problem in Go projects that share one Postgres server ([a survey of patterns](https://github.com/KolManis/go-integration-test-patterns)).

`internal/dbtest` implements it. A suite calls it once, in `BeforeSuite`:

```go
var _ = BeforeSuite(func(ctx SpecContext) {
	testPool = dbtest.NewPool(ctx, "web")
})
```

`NewPool` does the following:

1. Connects to the server named by `TEST_DATABASE_URL`, using its `postgres` maintenance database.
2. Creates a database with a random name, such as `noosphere_test_web_i3eawr5hexkow4f42atmwho53z`. The random part keeps two runs of the same package, for example from two terminals, apart as well.
3. Applies `db/migrations` to it with goose, so its schema is exactly what the migrations produce.
4. Returns a connection pool on it.

When the suite ends, the pool is closed and the database is dropped with `DROP DATABASE … WITH (FORCE)`, which also closes any connection a test left open. Creating and migrating a database took 26 to 54 ms in local measurements, once per package.

### Inside a package: rollback or truncate

Tests inside one Ginkgo suite run one after another, so they only need to clean up after each other. How depends on who commits.

**Database tests** (`internal/db`) call the code directly. Each test begins a transaction, runs everything inside it, and rolls it back at the end, so nothing is ever committed. When the code under test starts its own transaction, as sign-up does, it calls `Begin` on the test's transaction and gets a [savepoint](https://www.postgresql.org/docs/18/sql-savepoint.html) instead, which commits or rolls back inside the test's transaction.

**Handler tests** (`internal/web`) cannot work that way. The server takes the real connection pool, as it does in production, so its writes are committed. Changing the server to accept a test transaction instead would shape production code around the tests, so the tests adapt: each one starts by emptying every table except `goose_db_version`, which records the applied migrations:

```sql
TRUNCATE identities, identity_credentials, identity_credential_identifiers, registration_flows, sessions
```

The list of tables is read from `pg_tables`, so a table added by a new migration is included automatically. The tables are emptied at the start of each test rather than at the end, so a test always begins clean even if the previous run crashed halfway.

### How others do it

The same two ideas, a separate database where code runs in parallel and rollback or truncation inside it, appear across frameworks and projects:

| Who | What they do |
|---|---|
| [Django](https://docs.djangoproject.com/en/stable/topics/testing/tools/#transactiontestcase) | `TestCase` wraps each test in a transaction that is rolled back; `TransactionTestCase`, for code that commits, "resets the database after the test runs by truncating all tables". With [`--parallel`](https://docs.djangoproject.com/en/stable/ref/django-admin/), "each process gets its own database". |
| [Rails](https://guides.rubyonrails.org/testing.html#parallel-testing-with-processes) | Wraps each test in a transaction that is rolled back by default. Parallel workers each get their own database, named `test-database-0`, `test-database-1` and so on. |
| [Ory Kratos](https://github.com/ory/kratos/blob/b86338da04a040247a07f46100a86dcfb3875909/persistence/sql/persister_test.go) | Its persistence tests run `CREATE DATABASE testdb<uuid>` on a Postgres started for the tests, then apply the migrations. Most other tests use a fresh SQLite database ([`pkg/driver.go`](https://github.com/ory/kratos/blob/b86338da04a040247a07f46100a86dcfb3875909/pkg/driver.go)). |
| [Michael Stapelberg](https://michael.stapelberg.ch/posts/2024-11-19-testing-with-go-and-postgresql-ephemeral-dbs/) | One Postgres per `go test` run, and "a different PostgreSQL database for every test"; creating one takes 5 to 20 ms. |
| [Storj](https://storj.dev/blog/go-integration-tests-with-postgres) | Compares a container, a database and a schema per test: about 2 s, about 135 ms and about 25 ms per test in their measurements. Schemas are cheapest but "not as isolated as separate databases". |
| [A Go Forum thread](https://forum.golangbridge.org/t/running-go-test-caused-flaky-database-test-failures-due-to-concurrent-test-packages-sharing-a-single-mysql-test-database/42044) | The same flaky failures from packages sharing one test database; fixed by creating a database per package in setup and dropping it in teardown. |
| [golang/go#61318](https://github.com/golang/go/issues/61318) | An open proposal to run test packages one at a time without also serializing the build, because `-p 1` made one project's tests take 8 m 25 s instead of 50 s. |

### What this guarantees

- `go test ./...`, `make test`, an editor's test runner and CI all give the same result, whatever runs at the same time. Twenty parallel `go test ./...` runs and two simultaneous runs in separate terminals all passed.
- No test data reaches the development database, and no test database outlives its run.
- Tests inside one package still run one after another. Running a suite's tests in parallel, for example with `ginkgo -p`, would need a database per test.

## Ginkgo and Gomega

Specs are written with [Ginkgo](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/), because nested `Describe` and `It` blocks read as statements of behavior. Each package has one standard Go test function that starts its suite, so `go test` and editors run the specs like any other tests.

Spec names state the rule they prove, scoped exactly as the code or schema scopes it, for example "rejects an identifier already used by another credential of the same type". Test data is written inline and obviously fake (`test@example.com`, `"test-hash"`), and the cases are the ones a person would worry about: a different case, surrounding spaces, fullwidth letters, a decomposed `é`, an invisible character.

`make test` runs all tests with `go test`. `make test-v` runs them through the Ginkgo runner in random order, printing every spec, and accepts extra flags such as `ARGS='--focus SignUpWithPassword'`.

## Continuous integration

Every push to `main` and every pull request runs the GitHub Actions workflow `.github/workflows/ci.yml`. It starts a Postgres 18 service with the same credentials as the local database, checks that the code generated by sqlc matches the committed SQL (`make generate`, then `git diff --exit-code`), and runs `make check`.
