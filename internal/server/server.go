package server

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/Veysel440/go-notes-api/internal/config"
	"github.com/Veysel440/go-notes-api/internal/handlers"
	"github.com/Veysel440/go-notes-api/internal/logging"
	"github.com/Veysel440/go-notes-api/internal/metrics"
	"github.com/Veysel440/go-notes-api/internal/middleware"
	"github.com/Veysel440/go-notes-api/internal/openapi"
	"github.com/Veysel440/go-notes-api/internal/repos"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"
)

type Server struct {
	cfg config.Config
	db  *sql.DB
	mx  *metrics.Registry
	log *slog.Logger
}

func New(cfg config.Config, db *sql.DB) *Server {
	return &Server{
		cfg: cfg,
		db:  db,
		mx:  metrics.New(),
		log: logging.New(),
	}
}

func (s *Server) router() http.Handler {
	r := chi.NewRouter()

	rl := middleware.NewLimiter(rate.Limit(s.cfg.RateRPS), s.cfg.RateBurst, 5*time.Minute)

	r.Use(
		middleware.RequestID,
		chimw.RealIP,
		middleware.SecurityHeaders,
		middleware.CORS(s.cfg.CorsOrigins),
		middleware.BodyLimit(s.cfg.MaxBodyBytes),
		middleware.RecoverJSON(s.log),
		rl.Middleware,
		s.mx.MW,
		middleware.Logger(s.log),
		middleware.Audit{DB: s.db}.Middleware,
	)

	hh := handlers.Health{DB: s.db}
	r.Get("/healthz", hh.Live)
	r.Get("/readyz", hh.Ready)
	r.Get("/info", hh.Info)

	r.Group(func(gr chi.Router) {
		gr.Use(middleware.AllowCIDR(s.cfg.MetricsAllowCIDR))
		gr.Handle("/metrics", s.mx.Handler())
	})

	if s.cfg.Env != "prod" {
		r.Handle("/openapi.yaml", openapi.Spec())
		r.Handle("/docs", openapi.UI())
	}

	rolesRepo := &repos.Roles{DB: s.db}

	amx := repos.NewAuthMetrics(s.mx.Reg())
	emailLimiter := repos.NewEmailLimiter()
	au := handlers.Auth{
		Cfg:          s.cfg,
		Users:        &repos.Users{DB: s.db},
		Tokens:       &repos.RefreshTokens{DB: s.db},
		Roles:        rolesRepo,
		EmailLimiter: emailLimiter,
		Metrics:      amx,
	}
	r.Post("/auth/register", au.Register)
	r.Post("/auth/login", au.Login)
	r.Post("/auth/refresh", au.Refresh)

	r.Group(func(ar chi.Router) {
		ar.Use(middleware.Auth(), middleware.RequireRole(rolesRepo, "admin"))
		ar.Get("/admin/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ok":true}`))
		})
	})

	r.Group(func(ar chi.Router) {
		ar.Use(middleware.Auth(), middleware.RequireRole(rolesRepo, "admin"))

		ar.Get("/admin/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ok":true}`))
		})

		ua := handlers.AdminUsers{Cfg: s.cfg, Users: &repos.Users{DB: s.db}}
		ar.Get("/admin/users", ua.List)

		aroles := handlers.AdminRoles{Cfg: s.cfg, Roles: rolesRepo}
		ar.Post("/admin/users/{id}/roles", aroles.Post)

		aa := handlers.AdminAudit{Cfg: s.cfg, Audit: &repos.Audit{DB: s.db}}
		ar.Get("/admin/audit", aa.List) // ?from=RFC3339&to=RFC3339&limit=100&format=csv|json
	})

	nt := handlers.Notes{Repo: &repos.Notes{DB: s.db}}
	r.Route("/notes", func(pr chi.Router) {
		pr.Use(middleware.Auth(), middleware.RequireRole(rolesRepo, "user"))
		nt.Routes(pr)
	})

	return r
}

func (s *Server) HTTPServer() *http.Server {
	return &http.Server{
		Addr:         ":" + s.cfg.Port,
		Handler:      s.router(),
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
	}
}
