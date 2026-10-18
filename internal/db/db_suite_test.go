package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var testPool *pgxpool.Pool

func TestDB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DB Suite")
}

var _ = BeforeSuite(func(ctx SpecContext) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		Skip("TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(ctx, url)
	Expect(err).NotTo(HaveOccurred())
	testPool = pool
	DeferCleanup(pool.Close)
})

// testTx returns a transaction that is rolled back when the spec ends.
func testTx(ctx context.Context) pgx.Tx {
	GinkgoHelper()
	tx, err := testPool.Begin(ctx)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		tx.Rollback(context.Background())
	})
	return tx
}
