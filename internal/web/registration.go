package web

import (
	"errors"
	"net/http"
	"time"

	"github.com/Tianbo-Qiu/noosphere/internal/db"
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
			Title:   "Cookie required",
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
