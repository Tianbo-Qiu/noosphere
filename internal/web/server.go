package web

import (
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/Tianbo-Qiu/noosphere/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	// SecureCookies marks cookies Secure, so browsers send them only over
	// HTTPS. Turn it off only for local development over plain HTTP.
	SecureCookies bool
	// FlowLifetime is how long a registration flow stays valid.
	FlowLifetime time.Duration
}

type Server struct {
	cfg     Config
	pool    *pgxpool.Pool
	queries *db.Queries
	logger  *slog.Logger
	pages   map[string]*template.Template
}

func NewServer(cfg Config, pool *pgxpool.Pool, logger *slog.Logger) (*Server, error) {
	pages, err := parsePages()
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:     cfg,
		pool:    pool,
		queries: db.New(pool),
		logger:  logger,
		pages:   pages,
	}, nil
}

func (s *Server) requestLogger(r *http.Request) *slog.Logger {
	return s.logger.With("request_id", requestIDFromContext(r.Context()))
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(withRequestID)
	r.Use(s.withAccessLog)
	r.Use(middleware.RedirectSlashes)
	r.Handle("/static/*", http.FileServerFS(staticFS))
	r.Get("/registration", s.getRegistration)
	r.Post("/registration", s.postRegistration)
	r.Get("/registration/complete", s.getRegistrationComplete)
	return r
}
