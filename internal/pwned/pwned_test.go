package pwned

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client.Count", func() {
	It("returns how often a breached password was seen", func(ctx SpecContext) {
		fake := newFakeRangeAPI(map[string]int{"correct horse battery staple": 391})

		count, err := NewClient(fake.url, time.Second).Count(ctx, "correct horse battery staple")

		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(391))
	})

	It("returns 0 for a password that was never seen", func(ctx SpecContext) {
		fake := newFakeRangeAPI(map[string]int{"correct horse battery staple": 391})

		count, err := NewClient(fake.url, time.Second).Count(ctx, "quiet-otter-maps-the-river")

		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(BeZero())
	})

	It("sends only the first 5 characters of the SHA-1 hash, never the password", func(ctx SpecContext) {
		fake := newFakeRangeAPI(nil)

		_, err := NewClient(fake.url, time.Second).Count(ctx, "correct horse battery staple")

		Expect(err).NotTo(HaveOccurred())
		hash := sha1Hex("correct horse battery staple")
		Expect(fake.lastRequest().URL.String()).To(Equal("/range/" + hash[:5]))
	})

	It("asks for a padded response", func(ctx SpecContext) {
		fake := newFakeRangeAPI(nil)

		_, err := NewClient(fake.url, time.Second).Count(ctx, "correct horse battery staple")

		Expect(err).NotTo(HaveOccurred())
		Expect(fake.lastRequest().Header.Get("Add-Padding")).To(Equal("true"))
	})

	It("accepts a base URL with a trailing slash", func(ctx SpecContext) {
		fake := newFakeRangeAPI(map[string]int{"password": 52372427})

		count, err := NewClient(fake.url+"/", time.Second).Count(ctx, "password")

		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(52372427))
	})

	It("returns an error when the service answers with an error status", func(ctx SpecContext) {
		fake := newFakeRangeAPI(nil)
		fake.status = http.StatusServiceUnavailable

		_, err := NewClient(fake.url, time.Second).Count(ctx, "password")

		Expect(err).To(MatchError(ContainSubstring("unexpected status 503")))
	})

	It("returns an error when the service is slower than the timeout", func(ctx SpecContext) {
		fake := newFakeRangeAPI(nil)
		fake.delay = 300 * time.Millisecond

		_, err := NewClient(fake.url, 50*time.Millisecond).Count(ctx, "password")

		Expect(err).To(HaveOccurred())
	})
})

// fakeRangeAPI serves the Pwned Passwords range API for a fixed set of
// breached passwords, adds one padding entry to every response, and records
// the requests it receives.
type fakeRangeAPI struct {
	url    string
	counts map[string]int // full uppercase SHA-1 hash → times seen
	status int            // if set, answer with this status instead
	delay  time.Duration  // wait this long before answering

	mu       sync.Mutex
	requests []*http.Request
}

// newFakeRangeAPI starts a fake range API that knows the given passwords and
// how often each was seen. It stops when the spec ends.
func newFakeRangeAPI(seen map[string]int) *fakeRangeAPI {
	fake := &fakeRangeAPI{counts: map[string]int{}}
	for password, n := range seen {
		fake.counts[sha1Hex(password)] = n
	}
	ts := httptest.NewServer(fake)
	DeferCleanup(ts.Close)
	fake.url = ts.URL
	return fake
}

func (f *fakeRangeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r)
	f.mu.Unlock()

	time.Sleep(f.delay)
	if f.status != 0 {
		w.WriteHeader(f.status)
		return
	}
	prefix := strings.TrimPrefix(r.URL.Path, "/range/")
	for hash, n := range f.counts {
		if strings.HasPrefix(hash, prefix) {
			fmt.Fprintf(w, "%s:%d\r\n", hash[5:], n)
		}
	}
	fmt.Fprint(w, "00000000000000000000000000000000000:0\r\n") // padding entry
}

// lastRequest returns the most recent request the fake received.
func (f *fakeRangeAPI) lastRequest() *http.Request {
	GinkgoHelper()
	f.mu.Lock()
	defer f.mu.Unlock()
	Expect(f.requests).NotTo(BeEmpty())
	return f.requests[len(f.requests)-1]
}

// sha1Hex returns the password's SHA-1 hash in uppercase hex, as the range
// API uses it.
func sha1Hex(password string) string {
	sum := sha1.Sum([]byte(password))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}
