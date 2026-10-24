// Package dbtest gives each test package its own migrated database, so test
// packages that run in parallel never share tables. It is used only by tests.
package dbtest

import (
	"context"
	"crypto/rand"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pressly/goose/v3"
)

// NewPool creates a database for the calling test package on the server named
// by TEST_DATABASE_URL, applies db/migrations to it, and returns a pool on it.
// The pool is closed and the database dropped when the suite ends. Without
// TEST_DATABASE_URL, the suite is skipped.
//
// Call it from BeforeSuite. prefix names the database, for example "web"
// gives noosphere_test_web_<random>.
func NewPool(ctx context.Context, prefix string) *pgxpool.Pool {
	GinkgoHelper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		Skip("TEST_DATABASE_URL is not set")
	}
	server, err := url.Parse(raw)
	Expect(err).NotTo(HaveOccurred())

	admin, err := pgxpool.New(ctx, withDatabase(server, "postgres"))
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(admin.Close)

	name := "noosphere_test_" + prefix + "_" + strings.ToLower(rand.Text())
	_, err = admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(dropDatabase, admin, name)

	migrate(ctx, withDatabase(server, name))

	pool, err := pgxpool.New(ctx, withDatabase(server, name))
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(pool.Close)
	return pool
}

// withDatabase returns the server URL with its database name replaced.
func withDatabase(server *url.URL, name string) string {
	u := *server
	u.Path = "/" + name
	return u.String()
}

// dropDatabase drops the database, closing any connections still open to it.
func dropDatabase(ctx context.Context, admin *pgxpool.Pool, name string) {
	GinkgoHelper()
	_, err := admin.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
	Expect(err).NotTo(HaveOccurred())
}

// migrate applies db/migrations to the database at dbURL.
func migrate(ctx context.Context, dbURL string) {
	GinkgoHelper()
	conn, err := sql.Open("pgx", dbURL)
	Expect(err).NotTo(HaveOccurred())
	defer conn.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, conn, os.DirFS(migrationsDir()))
	Expect(err).NotTo(HaveOccurred())
	_, err = provider.Up(ctx)
	Expect(err).NotTo(HaveOccurred())
}

// migrationsDir returns the path of db/migrations, found from this file's
// location so it works from any test package.
func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "db", "migrations")
}
