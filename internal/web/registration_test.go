package web

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/Tianbo-Qiu/noosphere/internal/db"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GET /registration", func() {
	Context("without a flow", func() {
		It("creates a flow and redirects to it", func(ctx SpecContext) {
			ts := newTestServer(ctx)

			resp := get(newBrowser(), ts.URL+"/registration")

			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			flow := redirectedFlow(ctx, ts, resp)
			Expect(flow.ExpiresAt).To(BeTemporally("~", time.Now().Add(time.Hour), time.Minute))
		})

		It("sets an HttpOnly csrf cookie holding the flow's token", func(ctx SpecContext) {
			ts := newTestServer(ctx)

			resp := get(newBrowser(), ts.URL+"/registration")

			csrf := cookie(resp, csrfCookieName)
			Expect(csrf.Value).To(Equal(redirectedFlow(ctx, ts, resp).CsrfToken))
			Expect(csrf.HttpOnly).To(BeTrue())
			Expect(csrf.SameSite).To(Equal(http.SameSiteLaxMode))
			Expect(csrf.Path).To(Equal("/"))
		})

		It("reuses the browser's csrf token for its next flow", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			first := startFlow(ctx, ts, browser)

			resp := get(browser, ts.URL+"/registration")

			second := redirectedFlow(ctx, ts, resp)
			Expect(second.ID).NotTo(Equal(first.ID))
			Expect(second.CsrfToken).To(Equal(first.CsrfToken))
			Expect(resp.Cookies()).To(BeEmpty())
		})
	})

	Context("with a flow this browser started", func() {
		It("shows the registration form for that flow", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)

			resp := get(browser, ts.URL+"/registration?flow="+flow.ID.String())

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			page := body(resp)
			Expect(page).To(ContainSubstring(`<form method="post" action="/registration?flow=` + flow.ID.String() + `">`))
			Expect(page).To(ContainSubstring(`<input type="hidden" name="csrf_token" value="` + flow.CsrfToken + `">`))
			Expect(page).To(ContainSubstring(`name="email"`))
			Expect(page).To(ContainSubstring(`name="first_name"`))
			Expect(page).To(ContainSubstring(`name="last_name"`))
			Expect(page).To(ContainSubstring(`name="password"`))
		})
	})

	Context("with a flow that cannot be used", func() {
		It("starts a new flow when the flow ID is not a UUID", func(ctx SpecContext) {
			ts := newTestServer(ctx)

			resp := get(newBrowser(), ts.URL+"/registration?flow=not-a-uuid")

			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			redirectedFlow(ctx, ts, resp)
		})

		It("starts a new flow when no flow has that ID", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			missing := uuid.Must(uuid.NewV7())

			resp := get(newBrowser(), ts.URL+"/registration?flow="+missing.String())

			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			Expect(redirectedFlow(ctx, ts, resp).ID).NotTo(Equal(missing))
		})

		It("starts a new flow when the flow has expired", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			current := startFlow(ctx, ts, browser)
			expired, err := ts.Queries.CreateRegistrationFlow(ctx, db.CreateRegistrationFlowParams{
				CsrfToken: current.CsrfToken,
				ExpiresAt: time.Now().Add(-time.Minute),
			})
			Expect(err).NotTo(HaveOccurred())

			resp := get(browser, ts.URL+"/registration?flow="+expired.ID.String())

			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			Expect(redirectedFlow(ctx, ts, resp).ID).NotTo(Equal(expired.ID))
		})

		It("starts a new flow when the flow belongs to another browser", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			theirs := startFlow(ctx, ts, newBrowser())
			browser := newBrowser()
			ours := startFlow(ctx, ts, browser)

			resp := get(browser, ts.URL+"/registration?flow="+theirs.ID.String())

			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			flow := redirectedFlow(ctx, ts, resp)
			Expect(flow.ID).NotTo(Equal(theirs.ID))
			Expect(flow.CsrfToken).To(Equal(ours.CsrfToken))
		})
	})

	Context("from a browser that does not keep cookies", func() {
		It("shows the cookies-required page with 403 instead of redirecting again", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowserWithoutCookies()
			start := get(browser, ts.URL+"/registration")

			resp := get(browser, ts.URL+start.Header.Get("Location"))

			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
			Expect(body(resp)).To(ContainSubstring("<h1>Cookies required</h1>"))
		})
	})
})

var _ = Describe("POST /registration", func() {
	Context("with a valid flow and form", func() {
		It("creates the account and redirects to the account-created page", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)

			resp := submit(browser, ts, flow, url.Values{
				"csrf_token": {flow.CsrfToken},
				"email":      {"test@example.com"},
				"first_name": {"Test"},
				"last_name":  {"User"},
				"password":   {"correct horse battery staple"},
			})

			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			Expect(resp.Header.Get("Location")).To(Equal("/registration/complete"))
			account := storedAccount(ctx, ts, "test@example.com")
			Expect(account.Traits).To(Equal(db.Traits{Email: "test@example.com", FirstName: "Test", LastName: "User"}))
		})

		It("stores an Argon2id hash, never the password", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)

			submit(browser, ts, flow, validForm(flow, "test@example.com"))

			account := storedAccount(ctx, ts, "test@example.com")
			Expect(account.HashedPassword).To(HavePrefix("$argon2id$v=19$m=19456,t=2,p=1$"))
			Expect(account.HashedPassword).NotTo(ContainSubstring("correct horse battery staple"))
		})

		It("keeps the email's case in the profile, trims surrounding spaces, and stores the login identifier normalized", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)

			submit(browser, ts, flow, validForm(flow, " Test@Example.com\n"))

			account := storedAccount(ctx, ts, "test@example.com")
			Expect(account.Traits.Email).To(Equal("Test@Example.com"))
		})

		It("trims spaces around the names", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "test@example.com")
			form.Set("first_name", "  Test ")
			form.Set("last_name", "\tUser\n")

			submit(browser, ts, flow, form)

			Expect(storedAccount(ctx, ts, "test@example.com").Traits).To(Equal(db.Traits{Email: "test@example.com", FirstName: "Test", LastName: "User"}))
		})

		It("leaves out names that were not given", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)

			submit(browser, ts, flow, url.Values{
				"csrf_token": {flow.CsrfToken},
				"email":      {"test@example.com"},
				"password":   {"correct horse battery staple"},
			})

			Expect(storedAccount(ctx, ts, "test@example.com").Traits).To(Equal(db.Traits{Email: "test@example.com"}))
		})
	})

	Context("with a flow that cannot be used", func() {
		It("starts a new flow when the flow ID is not a UUID", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)

			resp := post(browser, ts.URL+"/registration?flow=not-a-uuid", validForm(flow, "test@example.com"))

			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			redirectedFlow(ctx, ts, resp)
			Expect(accountCount(ctx, ts)).To(BeZero())
		})

		It("starts a new flow when no flow has that ID", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			missing := uuid.Must(uuid.NewV7())

			resp := post(browser, ts.URL+"/registration?flow="+missing.String(), validForm(flow, "test@example.com"))

			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			Expect(redirectedFlow(ctx, ts, resp).ID).NotTo(Equal(missing))
			Expect(accountCount(ctx, ts)).To(BeZero())
		})

		It("starts a new flow when the flow has expired", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			current := startFlow(ctx, ts, browser)
			expired, err := ts.Queries.CreateRegistrationFlow(ctx, db.CreateRegistrationFlowParams{
				CsrfToken: current.CsrfToken,
				ExpiresAt: time.Now().Add(-time.Minute),
			})
			Expect(err).NotTo(HaveOccurred())

			resp := submit(browser, ts, expired, validForm(expired, "test@example.com"))

			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			Expect(redirectedFlow(ctx, ts, resp).ID).NotTo(Equal(expired.ID))
			Expect(accountCount(ctx, ts)).To(BeZero())
		})
	})

	Context("failing the CSRF check", func() {
		It("rejects a post without the csrf cookie", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			flow := startFlow(ctx, ts, newBrowser())

			resp := submit(newBrowserWithoutCookies(), ts, flow, validForm(flow, "test@example.com"))

			expectRejected(resp)
			Expect(accountCount(ctx, ts)).To(BeZero())
		})

		It("rejects a post whose cookie belongs to another browser", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			flow := startFlow(ctx, ts, newBrowser())
			other := newBrowser()
			startFlow(ctx, ts, other)

			resp := submit(other, ts, flow, validForm(flow, "test@example.com"))

			expectRejected(resp)
			Expect(accountCount(ctx, ts)).To(BeZero())
		})

		It("rejects a form without the csrf token", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "test@example.com")
			form.Del("csrf_token")

			resp := submit(browser, ts, flow, form)

			expectRejected(resp)
			Expect(accountCount(ctx, ts)).To(BeZero())
		})

		It("rejects a form with a different csrf token", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "test@example.com")
			form.Set("csrf_token", "ATTACKERTOKEN2345ABCDEFGHI")

			resp := submit(browser, ts, flow, form)

			expectRejected(resp)
			Expect(accountCount(ctx, ts)).To(BeZero())
		})
	})

	Context("with invalid input", func() {
		It("rejects an email that is already registered", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			first := newBrowser()
			firstFlow := startFlow(ctx, ts, first)
			submit(first, ts, firstFlow, validForm(firstFlow, "test@example.com"))
			second := newBrowser()
			secondFlow := startFlow(ctx, ts, second)

			resp := submit(second, ts, secondFlow, validForm(secondFlow, "Test@Example.com"))

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			Expect(accountCount(ctx, ts)).To(Equal(1))
		})

		It("rejects an email that cannot be normalized", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)

			resp := submit(browser, ts, flow, validForm(flow, "test\u200b@example.com")) // zero-width space after "test"

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			Expect(accountCount(ctx, ts)).To(BeZero())
		})

		It("rejects an empty password", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "test@example.com")
			form.Set("password", "")

			resp := submit(browser, ts, flow, form)

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			Expect(accountCount(ctx, ts)).To(BeZero())
		})
	})
})

var _ = Describe("GET /registration/complete", func() {
	It("shows the account-created page", func(ctx SpecContext) {
		ts := newTestServer(ctx)

		resp := get(newBrowser(), ts.URL+"/registration/complete")

		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(body(resp)).To(ContainSubstring("<h1>Account created</h1>"))
	})
})

// account is what sign-up stored for one login identifier.
type account struct {
	Traits         db.Traits
	HashedPassword string
}

// validForm returns a complete registration form for the flow, with the
// given email and the password "correct horse battery staple".
func validForm(flow db.RegistrationFlow, email string) url.Values {
	return url.Values{
		"csrf_token": {flow.CsrfToken},
		"email":      {email},
		"first_name": {"Test"},
		"last_name":  {"User"},
		"password":   {"correct horse battery staple"},
	}
}

// submit posts the registration form for the flow.
func submit(browser *http.Client, ts testServer, flow db.RegistrationFlow, form url.Values) *http.Response {
	GinkgoHelper()
	return post(browser, ts.URL+"/registration?flow="+flow.ID.String(), form)
}

// expectRejected checks that the response is the page for a failed CSRF check.
func expectRejected(resp *http.Response) {
	GinkgoHelper()
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	Expect(body(resp)).To(ContainSubstring("<h1>Request rejected</h1>"))
}

// storedAccount loads the traits and password hash stored for an identifier.
func storedAccount(ctx context.Context, ts testServer, identifier string) account {
	GinkgoHelper()
	var a account
	err := ts.Pool.QueryRow(ctx, `
		SELECT i.traits, c.config->>'hashed_password'
		FROM identity_credential_identifiers ci
		JOIN identity_credentials c ON c.id = ci.credential_id
		JOIN identities i ON i.id = c.identity_id
		WHERE ci.identifier = $1
		`, identifier).Scan(&a.Traits, &a.HashedPassword)
	Expect(err).NotTo(HaveOccurred(), "no account for %q", identifier)
	return a
}

// accountCount returns how many identities are stored.
func accountCount(ctx context.Context, ts testServer) int {
	GinkgoHelper()
	var n int
	Expect(ts.Pool.QueryRow(ctx, `SELECT count(*) FROM identities`).Scan(&n)).To(Succeed())
	return n
}

// startFlow opens the registration page in the browser and returns the flow
// it was redirected to.
func startFlow(ctx context.Context, ts testServer, browser *http.Client) db.RegistrationFlow {
	GinkgoHelper()
	resp := get(browser, ts.URL+"/registration")
	Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
	return redirectedFlow(ctx, ts, resp)
}

// redirectedFlow checks that the response redirects to the registration page
// for a stored flow, and returns that flow.
func redirectedFlow(ctx context.Context, ts testServer, resp *http.Response) db.RegistrationFlow {
	GinkgoHelper()
	location, err := url.Parse(resp.Header.Get("Location"))
	Expect(err).NotTo(HaveOccurred())
	Expect(location.Path).To(Equal("/registration"))

	id, err := uuid.Parse(location.Query().Get("flow"))
	Expect(err).NotTo(HaveOccurred(), "flow in %q is not a UUID", location)

	flow, err := ts.Queries.GetRegistrationFlow(ctx, id)
	Expect(err).NotTo(HaveOccurred(), "flow %s is not stored", id)
	return flow
}
