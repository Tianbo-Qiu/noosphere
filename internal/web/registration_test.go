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
		It("shows the cookie-required page with 403 instead of redirecting again", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowserWithoutCookies()
			start := get(browser, ts.URL+"/registration")

			resp := get(browser, ts.URL+start.Header.Get("Location"))

			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
			Expect(body(resp)).To(ContainSubstring("<h1>Cookie required</h1>"))
		})
	})
})

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
