package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Tianbo-Qiu/noosphere/internal/db"
	"github.com/Tianbo-Qiu/noosphere/internal/dbtest"
	"github.com/jackc/pgx/v5"
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
	testPool = dbtest.NewPool(ctx, "web")
})

// testServer is a running web server and queries on its database.
type testServer struct {
	URL     string
	Pool    *pgxpool.Pool
	Queries *db.Queries
}

// newTestServer empties the suite's database and starts the web server on it.
// The server commits for real, so each spec starts from empty tables instead
// of rolling back. Flows live for one hour.
func newTestServer(ctx context.Context) testServer {
	GinkgoHelper()
	truncateTables(ctx)

	srv, err := NewServer(Config{
		FlowLifetime: time.Hour,
	}, testPool, slog.New(slog.DiscardHandler))
	Expect(err).NotTo(HaveOccurred())

	ts := httptest.NewServer(srv.Handler())
	DeferCleanup(ts.Close)

	return testServer{URL: ts.URL, Pool: testPool, Queries: db.New(testPool)}
}

// truncateTables deletes every row from every table except goose's record
// of applied migrations.
func truncateTables(ctx context.Context) {
	GinkgoHelper()
	rows, err := testPool.Query(ctx, `
		SELECT tablename FROM pg_tables
		WHERE schemaname = 'public' AND tablename <> 'goose_db_version'
		`)
	Expect(err).NotTo(HaveOccurred())
	tables, err := pgx.CollectRows(rows, pgx.RowTo[string])
	Expect(err).NotTo(HaveOccurred())

	names := make([]string, len(tables))
	for i, t := range tables {
		names[i] = pgx.Identifier{t}.Sanitize()
	}
	_, err = testPool.Exec(ctx, "TRUNCATE "+strings.Join(names, ", "))
	Expect(err).NotTo(HaveOccurred())
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

// post sends a form as a POST request and returns the response.
func post(client *http.Client, target string, form url.Values) *http.Response {
	GinkgoHelper()
	resp, err := client.PostForm(target, form)
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
