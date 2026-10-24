package web

import (
	"errors"
	"net/http"
	"time"

	"github.com/Tianbo-Qiu/noosphere/internal/db"
	"github.com/Tianbo-Qiu/noosphere/internal/password"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// registrationPage is the data the registration template renders.
type registrationPage struct {
	Title     string
	Action    string
	CSRFToken string
	Email     string
	FirstName string
	LastName  string
}

// errorPage is the data the error template renders
type errorPage struct {
	Title   string
	Message string
}

func (s *Server) getRegistration(w http.ResponseWriter, r *http.Request) {
	log := s.requestLogger(r)

	rawID := r.URL.Query().Get("flow")
	id, err := uuid.Parse(rawID)
	if err != nil {
		log.Info("no valid registration flow in request", "flow", rawID, "err", err)
		s.startRegistration(w, r)
		return
	}

	flow, err := s.queries.GetRegistrationFlow(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Info("registration flow not found", "flow_id", id)
			s.startRegistration(w, r)
			return
		}
		log.Error("get registration flow", "flow_id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		log.Warn("registration flow requested without csrf cookie", "flow_id", flow.ID)
		s.render(w, http.StatusForbidden, "error", errorPage{
			Title:   "Cookies required",
			Message: "Your browser did not send back the security cookie. Allow cookies for this site and start again.",
		})
		return
	}
	if cookie.Value != flow.CsrfToken {
		log.Warn("registration flow belongs to another browser", "flow_id", flow.ID)
		s.startRegistration(w, r)
		return
	}
	if time.Now().After(flow.ExpiresAt) {
		log.Info("registration flow expired", "flow_id", flow.ID, "expires_at", flow.ExpiresAt)
		s.startRegistration(w, r)
		return
	}

	log.Info("registration flow shown", "flow_id", flow.ID)
	s.render(w, http.StatusOK, "registration", registrationPage{
		Title:     "Sign up",
		Action:    "/registration?flow=" + flow.ID.String(),
		CSRFToken: flow.CsrfToken,
	})
}

func (s *Server) startRegistration(w http.ResponseWriter, r *http.Request) {
	log := s.requestLogger(r)

	flow, err := s.queries.CreateRegistrationFlow(r.Context(), db.CreateRegistrationFlowParams{
		CsrfToken: s.csrfToken(w, r),
		ExpiresAt: time.Now().Add(s.cfg.FlowLifetime),
	})
	if err != nil {
		log.Error("create registration flow", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	log.Info("registration flow created", "flow_id", flow.ID)
	http.Redirect(w, r, "/registration?flow="+flow.ID.String(), http.StatusSeeOther)
}

// postRegistration creates an account from the submitted registration form.
func (s *Server) postRegistration(w http.ResponseWriter, r *http.Request) {
	log := s.requestLogger(r)

	rawID := r.URL.Query().Get("flow")
	id, err := uuid.Parse(rawID)
	if err != nil {
		log.Info("registration submitted without a valid flow", "flow", rawID, "err", err)
		s.startRegistration(w, r)
		return
	}

	flow, err := s.queries.GetRegistrationFlow(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Info("registration submitted for an unknown flow", "flow_id", id)
			s.startRegistration(w, r)
			return
		}
		log.Error("get registration flow", "flow_id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if time.Now().After(flow.ExpiresAt) {
		log.Info("registration submitted for an expired flow", "flow_id", flow.ID, "expires_at", flow.ExpiresAt)
		s.startRegistration(w, r)
		return
	}

	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value != flow.CsrfToken {
		log.Warn("registration rejected: csrf cookie missing or does not match the flow", "flow_id", flow.ID)
		s.renderRejected(w)
		return
	}
	if r.PostFormValue("csrf_token") != flow.CsrfToken {
		log.Warn("registration rejected: form csrf token does not match the flow", "flow_id", flow.ID)
		s.renderRejected(w)
		return
	}

	hashed, err := password.Hash(r.PostFormValue("password"))
	if err != nil {
		log.Info("registration rejected: invalid password", "flow_id", flow.ID, "err", err)
		http.Error(w, "invalid password", http.StatusBadRequest)
		return
	}

	email := r.PostFormValue("email")
	traits := db.Traits{
		Email:     email,
		FirstName: r.PostFormValue("first_name"),
		LastName:  r.PostFormValue("last_name"),
	}
	identity, err := db.CreatePasswordIdentity(r.Context(), s.pool, traits, email, db.PasswordConfig{HashedPassword: hashed})
	if errors.Is(err, db.ErrInvalidIdentifier) || errors.Is(err, db.ErrIdentifierTaken) {
		log.Info("registration rejected: email invalid or taken", "flow_id", flow.ID, "err", err)
		http.Error(w, "email invalid or already registered", http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Error("create identity", "flow_id", flow.ID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	log.Info("identity created", "flow_id", flow.ID, "identity_id", identity.ID)
	http.Redirect(w, r, "/registration/complete", http.StatusSeeOther)
}

// renderRejected shows the page for a form post that failed the CSRF check.
func (s *Server) renderRejected(w http.ResponseWriter) {
	s.render(w, http.StatusForbidden, "error", errorPage{
		Title:   "Request rejected",
		Message: "This form could not be verified as coming from this browser. Start again.",
	})
}

// registrationCompletePage is the data the registration_complete template renders.
type registrationCompletePage struct {
	Title string
}

// getRegistrationComplete shows the page users see after creating an account.
func (s *Server) getRegistrationComplete(w http.ResponseWriter, r *http.Request) {
	s.render(w, http.StatusOK, "registration_complete", registrationCompletePage{Title: "Account created"})
}
