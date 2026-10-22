package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Tianbo-Qiu/noosphere/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWeb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Web Suite")
}

var testPool *pgxpool.Pool

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

// testServer is a running web server and the queries it uses.
type testServer struct {
	URL     string
	Queries *db.Queries
}

// newTestServer starts the web server on a transaction that is rolled back
// when the spec ends. Flows live for one hour.
func newTestServer(ctx context.Context) testServer {
	GinkgoHelper()
	tx, err := testPool.Begin(ctx)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		tx.Rollback(context.Background())
	})

	queries := db.New(tx)
	srv, err := NewServer(Config{
		FlowLifetime: time.Hour,
	}, queries, slog.New(slog.DiscardHandler))
	Expect(err).NotTo(HaveOccurred())

	ts := httptest.NewServer(srv.Handler())
	DeferCleanup(ts.Close)

	return testServer{URL: ts.URL, Queries: queries}
}

// newBrowser returns a client that keeps cookies like a browser and does not
// follow redirects, so specs can check each response.
func newBrowser() *http.Client {
	GinkgoHelper()
	jar, err := cookiejar.New(nil)
	Expect(err).NotTo(HaveOccurred())
	return &http.Client{Jar: jar, CheckRedirect: doNotFollowRedirects}
}

// newBrowserWithoutCookies returns a client that, like a browser with cookies
// blocked, never sends cookies back.
func newBrowserWithoutCookies() *http.Client {
	return &http.Client{CheckRedirect: doNotFollowRedirects}
}

// doNotFollowRedirects makes the client return a redirect response as is.
func doNotFollowRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// get sends a GET request and returns the response.
func get(client *http.Client, url string) *http.Response {
	GinkgoHelper()
	resp, err := client.Get(url)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(resp.Body.Close)
	return resp
}

// body reads the whole response body.
func body(resp *http.Response) string {
	GinkgoHelper()
	b, err := io.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	return string(b)
}

// cookie returns the cookie the response sets with the given name.
func cookie(resp *http.Response, name string) *http.Cookie {
	GinkgoHelper()
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c
		}
	}
	Fail("response sets no " + name + " cookie")
	return nil
}
