package web

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Tianbo-Qiu/noosphere/internal/db"
	"github.com/Tianbo-Qiu/noosphere/internal/identifier"
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
	Errors    db.RegistrationErrors
}

// errorPage is the data the error template renders.
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
		s.renderInternalError(w)
		return
	}

	if time.Now().After(flow.ExpiresAt) {
		log.Info("registration flow expired", "flow_id", flow.ID, "expires_at", flow.ExpiresAt)
		s.startRegistration(w, r)
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

	log.Info("registration flow shown", "flow_id", flow.ID)
	s.render(w, http.StatusOK, "registration", registrationPage{
		Title:     "Sign up",
		Action:    "/registration?flow=" + flow.ID.String(),
		CSRFToken: flow.CsrfToken,
		Email:     flow.State.Traits.Email,
		FirstName: flow.State.Traits.FirstName,
		LastName:  flow.State.Traits.LastName,
		Errors:    flow.State.Errors,
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
		s.renderInternalError(w)
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
		s.renderInternalError(w)
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

	email := strings.TrimSpace(r.PostFormValue("email"))
	traits := db.Traits{
		Email:     email,
		FirstName: strings.TrimSpace(r.PostFormValue("first_name")),
		LastName:  strings.TrimSpace(r.PostFormValue("last_name")),
	}
	plain := r.PostFormValue("password")

	var formErrors db.RegistrationErrors
	if _, err := identifier.Normalize(email); err != nil {
		formErrors.Email = "Enter a valid email address."
	}
	if plain == "" {
		formErrors.Password = "Enter a password."
	}
	if !formErrors.IsZero() {
		log.Info("registration rejected: invalid input", "flow_id", flow.ID, "form_errors", formErrors)
		s.saveFormErrorsAndRedirect(w, r, log, flow.ID, db.RegistrationFlowState{
			Traits: traits,
			Errors: formErrors,
		})
		return
	}

	hashed, err := password.Hash(plain)
	if err != nil {
		formErrors.Password = "This password contains a character that cannot be used."
		log.Info("registration rejected: password cannot be normalized", "flow_id", flow.ID, "form_errors", formErrors, "err", err)
		s.saveFormErrorsAndRedirect(w, r, log, flow.ID, db.RegistrationFlowState{
			Traits: traits,
			Errors: formErrors,
		})
		return
	}

	identity, err := db.CreatePasswordIdentity(r.Context(), s.pool, traits, email, db.PasswordConfig{HashedPassword: hashed})
	if errors.Is(err, db.ErrIdentifierTaken) {
		formErrors.Email = "An account with this email already exists."
		log.Info("registration rejected: email already registered", "flow_id", flow.ID, "form_errors", formErrors)
		s.saveFormErrorsAndRedirect(w, r, log, flow.ID, db.RegistrationFlowState{
			Traits: traits,
			Errors: formErrors,
		})
		return
	}
	if err != nil {
		log.Error("create identity", "flow_id", flow.ID, "err", err)
		s.renderInternalError(w)
		return
	}
	log.Info("identity created", "flow_id", flow.ID, "identity_id", identity.ID)

	if err := s.queries.DeleteRegistrationFlow(r.Context(), flow.ID); err != nil {
		log.Error("delete registration flow", "flow_id", flow.ID, "err", err)
	}
	http.Redirect(w, r, "/registration/complete", http.StatusSeeOther)
}

// renderInternalError shows the page for a failure the user cannot fix. The
// cause is logged by the caller and never shown.
func (s *Server) renderInternalError(w http.ResponseWriter) {
	s.render(w, http.StatusInternalServerError, "error", errorPage{
		Title:   "Something went wrong",
		Message: "We could not complete your request. Try again in a moment.",
	})
}

// renderRejected shows the page for a form post that failed the CSRF check.
func (s *Server) renderRejected(w http.ResponseWriter) {
	s.render(w, http.StatusForbidden, "error", errorPage{
		Title:   "Request rejected",
		Message: "This form could not be verified as coming from this browser. Start again.",
	})
}

// saveFormErrorsAndRedirect stores what the user typed and the errors in the
// flow, then redirects back to the form.
func (s *Server) saveFormErrorsAndRedirect(w http.ResponseWriter, r *http.Request, log *slog.Logger, flowID uuid.UUID, state db.RegistrationFlowState) {
	err := s.queries.UpdateRegistrationFlowState(r.Context(), db.UpdateRegistrationFlowStateParams{
		ID:    flowID,
		State: state,
	})
	if err != nil {
		log.Error("save registration flow state", "flow_id", flowID, "err", err)
		s.renderInternalError(w)
		return
	}
	http.Redirect(w, r, "/registration?flow="+flowID.String(), http.StatusSeeOther)
}

// registrationCompletePage is the data the registration_complete template renders.
type registrationCompletePage struct {
	Title string
}

// getRegistrationComplete shows the page users see after creating an account.
func (s *Server) getRegistrationComplete(w http.ResponseWriter, r *http.Request) {
	s.render(w, http.StatusOK, "registration_complete", registrationCompletePage{Title: "Account created"})
}
