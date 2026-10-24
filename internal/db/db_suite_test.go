package db

import (
	"context"
	"testing"

	"github.com/Tianbo-Qiu/noosphere/internal/dbtest"
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
	testPool = dbtest.NewPool(ctx, "db")
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
