package web

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Tianbo-Qiu/noosphere/internal/db"
	"github.com/Tianbo-Qiu/noosphere/internal/pwned"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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

		It("marks the csrf cookie Secure when the server is configured for HTTPS", func(ctx SpecContext) {
			ts := newTestServerWithConfig(ctx, Config{SecureCookies: true, FlowLifetime: time.Hour})

			resp := get(newBrowser(), ts.URL+"/registration")

			Expect(cookie(resp, csrfCookieName).Secure).To(BeTrue())
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
			Expect(page).To(ContainSubstring(`<label for="first_name">First name (optional)</label>`))
			Expect(page).To(ContainSubstring(`<label for="last_name">Last name (optional)</label>`))
			Expect(page).To(ContainSubstring(`<small id="password-hint">Use 8 to 64 characters.</small>`))
			Expect(page).To(ContainSubstring(`spellcheck="false" autocapitalize="none" data-password-toggle`))
			Expect(page).To(ContainSubstring(`<script src="/static/password-toggle.js" defer></script>`))
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

	Context("with input the user can fix", func() {
		It("shows the form again with an error when the email is already registered", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			first := newBrowser()
			firstFlow := startFlow(ctx, ts, first)
			submit(first, ts, firstFlow, validForm(firstFlow, "test@example.com"))
			second := newBrowser()
			secondFlow := startFlow(ctx, ts, second)

			resp := submit(second, ts, secondFlow, validForm(secondFlow, "Test@Example.com"))

			page := followBackToForm(second, ts, secondFlow, resp)
			Expect(page).To(ContainSubstring(`<small id="email-error">An account with this email already exists.</small>`))
			Expect(accountCount(ctx, ts)).To(Equal(1))
		})

		It("shows the form again with an error when the email cannot be normalized", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)

			resp := submit(browser, ts, flow, validForm(flow, "test\u200b@example.com")) // zero-width space after "test"

			page := followBackToForm(browser, ts, flow, resp)
			Expect(page).To(ContainSubstring(`<small id="email-error">Enter a valid email address.</small>`))
			Expect(accountCount(ctx, ts)).To(BeZero())
		})

		It("shows the form again with an error when the password is empty", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "test@example.com")
			form.Set("password", "")

			resp := submit(browser, ts, flow, form)

			page := followBackToForm(browser, ts, flow, resp)
			Expect(page).To(ContainSubstring(`<small id="password-error">Enter a password.</small>`))
			Expect(accountCount(ctx, ts)).To(BeZero())
		})

		DescribeTable("shows the form again with an error when the password length is outside 8 to 64 characters",
			func(ctx SpecContext, plain, message string) {
				ts := newTestServer(ctx)
				browser := newBrowser()
				flow := startFlow(ctx, ts, browser)
				form := validForm(flow, "test@example.com")
				form.Set("password", plain)

				resp := submit(browser, ts, flow, form)

				page := followBackToForm(browser, ts, flow, resp)
				Expect(page).To(ContainSubstring(`<small id="password-error">` + message + `</small>`))
				Expect(accountCount(ctx, ts)).To(BeZero())
			},
			Entry("7 characters", "1234567", "Use at least 8 characters."),
			Entry("65 characters", strings.Repeat("a", 65), "Use at most 64 characters."),
		)

		DescribeTable("accepts a password at the length limits",
			func(ctx SpecContext, plain string) {
				ts := newTestServer(ctx)
				browser := newBrowser()
				flow := startFlow(ctx, ts, browser)
				form := validForm(flow, "test@example.com")
				form.Set("password", plain)

				resp := submit(browser, ts, flow, form)

				Expect(resp.Header.Get("Location")).To(Equal("/registration/complete"))
				Expect(accountCount(ctx, ts)).To(Equal(1))
			},
			Entry("8 characters", "12345678"),
			Entry("64 characters", strings.Repeat("a", 64)),
		)

		It("shows the form again with an error when the password has a character that cannot be used", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "test@example.com")
			form.Set("password", "secret\u0007") // BEL, a control character

			resp := submit(browser, ts, flow, form)

			page := followBackToForm(browser, ts, flow, resp)
			Expect(page).To(ContainSubstring(`<small id="password-error">This password contains a character that cannot be used.</small>`))
			Expect(accountCount(ctx, ts)).To(BeZero())
		})

		It("shows the email and password errors together", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "")
			form.Set("password", "")

			resp := submit(browser, ts, flow, form)

			page := followBackToForm(browser, ts, flow, resp)
			Expect(page).To(ContainSubstring(`<small id="email-error">Enter a valid email address.</small>`))
			Expect(page).To(ContainSubstring(`<small id="password-error">Enter a password.</small>`))
		})

		It("marks the invalid field and links it to its error", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "test@example.com")
			form.Set("password", "")

			resp := submit(browser, ts, flow, form)

			page := followBackToForm(browser, ts, flow, resp)
			Expect(page).To(ContainSubstring(`aria-describedby="password-error" aria-invalid="true">`))
			Expect(page).NotTo(ContainSubstring(`aria-describedby="email-error"`))
			Expect(page).NotTo(ContainSubstring(`id="password-hint"`), "the error replaces the hint")
		})

		It("keeps what the user typed, except the password", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "Test@Example.com")
			form.Set("password", "secret\u0007") // BEL, a control character

			resp := submit(browser, ts, flow, form)

			page := followBackToForm(browser, ts, flow, resp)
			Expect(page).To(ContainSubstring(`name="email" type="email" value="Test@Example.com"`))
			Expect(page).To(ContainSubstring(`name="first_name" type="text" value="Test"`))
			Expect(page).To(ContainSubstring(`name="last_name" type="text" value="User"`))
			Expect(page).NotTo(ContainSubstring("secret"))
		})
	})

	Context("with the context check on and a password that is a word from the sign-up itself", func() {
		DescribeTable("shows the form again with an error",
			func(ctx SpecContext, email, firstName, plain string) {
				ts := newTestServerWithConfig(ctx, Config{FlowLifetime: time.Hour, PasswordContextCheck: true})
				browser := newBrowser()
				flow := startFlow(ctx, ts, browser)
				form := validForm(flow, email)
				form.Set("first_name", firstName)
				form.Set("password", plain)

				resp := submit(browser, ts, flow, form)

				page := followBackToForm(browser, ts, flow, resp)
				Expect(page).To(ContainSubstring(`<small id="password-error">Don&#39;t use your email, your name or the site&#39;s name as your password.</small>`))
				Expect(accountCount(ctx, ts)).To(BeZero())
			},
			Entry("the email", "test@example.com", "Test", "Test@Example.com"),
			Entry("the part of the email before the @", "longname@example.com", "Test", "LongName"),
			Entry("the first name", "test@example.com", "Bartholomew", "bartholomew"),
			Entry("the site's name", "test@example.com", "Test", "Noosphere"),
		)

		It("accepts a password that only contains the email", func(ctx SpecContext) {
			ts := newTestServerWithConfig(ctx, Config{FlowLifetime: time.Hour, PasswordContextCheck: true})
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "test@example.com")
			form.Set("password", "test@example.com and a lot more")

			resp := submit(browser, ts, flow, form)

			Expect(resp.Header.Get("Location")).To(Equal("/registration/complete"))
		})

		It("does not contact the breached password service", func(ctx SpecContext) {
			hibp := newFakePwnedPasswords(nil)
			ts := newTestServerWithConfig(ctx, Config{FlowLifetime: time.Hour, PasswordContextCheck: true, PwnedPasswords: pwned.NewClient(hibp.url, time.Second)})
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "test@example.com")
			form.Set("password", "noosphere")

			submit(browser, ts, flow, form)

			Expect(hibp.requestCount()).To(BeZero())
		})
	})

	Context("with the context check off", func() {
		It("accepts a password that is the site's name", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "test@example.com")
			form.Set("password", "noosphere")

			resp := submit(browser, ts, flow, form)

			Expect(resp.Header.Get("Location")).To(Equal("/registration/complete"))
		})
	})

	Context("with the breached password check on", func() {
		It("shows the form again with an error when the password has been in a breach", func(ctx SpecContext) {
			hibp := newFakePwnedPasswords(map[string]int{"correct horse battery staple": 391})
			ts := newTestServerWithConfig(ctx, Config{FlowLifetime: time.Hour, PwnedPasswords: pwned.NewClient(hibp.url, time.Second)})
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)

			resp := submit(browser, ts, flow, validForm(flow, "test@example.com"))

			page := followBackToForm(browser, ts, flow, resp)
			Expect(page).To(ContainSubstring(`<small id="password-error">This password has appeared in a data breach. Choose a different one.</small>`))
			Expect(accountCount(ctx, ts)).To(BeZero())
		})

		It("creates the account when the password was never in a breach", func(ctx SpecContext) {
			hibp := newFakePwnedPasswords(map[string]int{"password": 52372427})
			ts := newTestServerWithConfig(ctx, Config{FlowLifetime: time.Hour, PwnedPasswords: pwned.NewClient(hibp.url, time.Second)})
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)

			resp := submit(browser, ts, flow, validForm(flow, "test@example.com"))

			Expect(resp.Header.Get("Location")).To(Equal("/registration/complete"))
			Expect(hibp.requestCount()).To(Equal(1))
		})

		It("creates the account when the service fails, so sign-up never depends on it", func(ctx SpecContext) {
			hibp := newFakePwnedPasswords(map[string]int{"correct horse battery staple": 391})
			hibp.status = http.StatusServiceUnavailable
			ts := newTestServerWithConfig(ctx, Config{FlowLifetime: time.Hour, PwnedPasswords: pwned.NewClient(hibp.url, time.Second)})
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)

			resp := submit(browser, ts, flow, validForm(flow, "test@example.com"))

			Expect(resp.Header.Get("Location")).To(Equal("/registration/complete"))
			Expect(accountCount(ctx, ts)).To(Equal(1))
		})

		It("creates the account when the service is slower than the timeout", func(ctx SpecContext) {
			hibp := newFakePwnedPasswords(map[string]int{"correct horse battery staple": 391})
			hibp.delay = 300 * time.Millisecond
			ts := newTestServerWithConfig(ctx, Config{FlowLifetime: time.Hour, PwnedPasswords: pwned.NewClient(hibp.url, 50*time.Millisecond)})
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)

			resp := submit(browser, ts, flow, validForm(flow, "test@example.com"))

			Expect(resp.Header.Get("Location")).To(Equal("/registration/complete"))
			Expect(accountCount(ctx, ts)).To(Equal(1))
		})

		It("does not contact the service for a password that is already invalid", func(ctx SpecContext) {
			hibp := newFakePwnedPasswords(nil)
			ts := newTestServerWithConfig(ctx, Config{FlowLifetime: time.Hour, PwnedPasswords: pwned.NewClient(hibp.url, time.Second)})
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "test@example.com")
			form.Set("password", "1234567")

			submit(browser, ts, flow, form)

			Expect(hibp.requestCount()).To(BeZero())
		})
	})

	Context("with the breached password check off, the default", func() {
		It("accepts a breached password without contacting any service", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "test@example.com")
			form.Set("password", "password") // seen 52,372,427 times in breaches

			resp := submit(browser, ts, flow, form)

			Expect(resp.Header.Get("Location")).To(Equal("/registration/complete"))
		})
	})

	Context("after fixing a mistake", func() {
		It("creates the account from the same flow and drops the old errors", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			form := validForm(flow, "test@example.com")
			form.Set("password", "")
			followBackToForm(browser, ts, flow, submit(browser, ts, flow, form))

			resp := submit(browser, ts, flow, validForm(flow, "test@example.com"))

			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			Expect(resp.Header.Get("Location")).To(Equal("/registration/complete"))
			Expect(storedAccount(ctx, ts, "test@example.com").Traits.Email).To(Equal("test@example.com"))
		})
	})

	Context("when two people sign up with the same email at the same moment", func() {
		It("creates one account and tells the other the email is taken", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			first, second := newBrowser(), newBrowser()
			firstFlow, secondFlow := startFlow(ctx, ts, first), startFlow(ctx, ts, second)

			responses := submitAtOnce(
				submission{first, ts.URL + "/registration?flow=" + firstFlow.ID.String(), validForm(firstFlow, "test@example.com")},
				submission{second, ts.URL + "/registration?flow=" + secondFlow.ID.String(), validForm(secondFlow, "Test@Example.com")},
			)

			locations := []string{responses[0].Header.Get("Location"), responses[1].Header.Get("Location")}
			Expect(locations).To(ConsistOf(
				"/registration/complete",
				Or(Equal("/registration?flow="+firstFlow.ID.String()), Equal("/registration?flow="+secondFlow.ID.String())),
			))
			Expect(accountCount(ctx, ts)).To(Equal(1))
		})
	})

	Context("when the database is unavailable", func() {
		It("shows the something-went-wrong page with 500 and no details", func(ctx SpecContext) {
			truncateTables(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, newTestServer(ctx), browser)
			ts := startServer(Config{FlowLifetime: time.Hour}, closedPool(ctx))

			resp := submit(browser, ts, flow, validForm(flow, "test@example.com"))

			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
			page := body(resp)
			Expect(page).To(ContainSubstring("<h1>Something went wrong</h1>"))
			Expect(page).NotTo(ContainSubstring("closed pool"))
		})
	})

	Context("after the account is created", func() {
		It("deletes the flow, so its URL starts a new sign-up", func(ctx SpecContext) {
			ts := newTestServer(ctx)
			browser := newBrowser()
			flow := startFlow(ctx, ts, browser)
			submit(browser, ts, flow, validForm(flow, "test@example.com"))

			resp := get(browser, ts.URL+"/registration?flow="+flow.ID.String())

			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			Expect(redirectedFlow(ctx, ts, resp).ID).NotTo(Equal(flow.ID))
		})
	})
})

var _ = Describe("every response", func() {
	It("carries its own request ID in X-Request-ID", func(ctx SpecContext) {
		ts := newTestServer(ctx)
		browser := newBrowser()

		first := get(browser, ts.URL+"/registration")
		second := get(browser, ts.URL+"/registration/complete")

		Expect(first.Header.Get("X-Request-ID")).NotTo(BeEmpty())
		Expect(second.Header.Get("X-Request-ID")).NotTo(BeEmpty())
		Expect(second.Header.Get("X-Request-ID")).NotTo(Equal(first.Header.Get("X-Request-ID")))
	})
})

var _ = Describe("GET /static/password-toggle.js", func() {
	It("serves the script that adds the Show button", func(ctx SpecContext) {
		ts := newTestServer(ctx)

		resp := get(newBrowser(), ts.URL+"/static/password-toggle.js")

		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(resp.Header.Get("Content-Type")).To(HavePrefix("text/javascript"))
		Expect(body(resp)).To(ContainSubstring("data-password-toggle"))
	})
})

var _ = Describe("a page URL with a trailing slash", func() {
	It("redirects to the URL without it", func(ctx SpecContext) {
		ts := newTestServer(ctx)

		resp := get(newBrowser(), ts.URL+"/registration/")

		Expect(resp.StatusCode).To(Equal(http.StatusMovedPermanently))
		Expect(resp.Header.Get("Location")).To(Equal("/registration"))
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

// followBackToForm checks that the response redirects back to the flow's form,
// opens it, and returns the page.
func followBackToForm(browser *http.Client, ts testServer, flow db.RegistrationFlow, resp *http.Response) string {
	GinkgoHelper()
	Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
	Expect(resp.Header.Get("Location")).To(Equal("/registration?flow=" + flow.ID.String()))

	form := get(browser, ts.URL+resp.Header.Get("Location"))
	Expect(form.StatusCode).To(Equal(http.StatusOK))
	return body(form)
}

// fakePwnedPasswords serves the Pwned Passwords range API for a fixed set of
// breached passwords, and counts the requests it receives.
type fakePwnedPasswords struct {
	url    string
	counts map[string]int // full uppercase SHA-1 hash → times seen
	status int            // if set, answer with this status instead
	delay  time.Duration  // wait this long before answering

	requests atomic.Int32
}

// newFakePwnedPasswords starts a fake range API that knows the given
// passwords and how often each was seen. It stops when the spec ends.
func newFakePwnedPasswords(seen map[string]int) *fakePwnedPasswords {
	fake := &fakePwnedPasswords{counts: map[string]int{}}
	for password, n := range seen {
		sum := sha1.Sum([]byte(password))
		fake.counts[strings.ToUpper(hex.EncodeToString(sum[:]))] = n
	}
	ts := httptest.NewServer(fake)
	DeferCleanup(ts.Close)
	fake.url = ts.URL
	return fake
}

func (f *fakePwnedPasswords) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.requests.Add(1)
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
}

// requestCount returns how many requests the fake has received.
func (f *fakePwnedPasswords) requestCount() int {
	return int(f.requests.Load())
}

// submission is one form post: who sends it, where, and what.
type submission struct {
	browser *http.Client
	url     string
	form    url.Values
}

// submitAtOnce sends all submissions at the same time and returns their
// responses in the same order.
func submitAtOnce(submissions ...submission) []*http.Response {
	GinkgoHelper()
	responses := make([]*http.Response, len(submissions))
	errs := make([]error, len(submissions))
	var wg sync.WaitGroup
	for i, sub := range submissions {
		wg.Go(func() {
			responses[i], errs[i] = sub.browser.PostForm(sub.url, sub.form)
		})
	}
	wg.Wait()
	for i := range submissions {
		Expect(errs[i]).NotTo(HaveOccurred())
		DeferCleanup(responses[i].Body.Close)
	}
	return responses
}

// closedPool returns a pool that is already closed, so every query on it fails
// the way it would if the database were unreachable.
func closedPool(ctx context.Context) *pgxpool.Pool {
	GinkgoHelper()
	pool, err := pgxpool.New(ctx, testPool.Config().ConnString())
	Expect(err).NotTo(HaveOccurred())
	pool.Close()
	return pool
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
