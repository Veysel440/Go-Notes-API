package server

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/Veysel440/go-notes-api/internal/config"
	"github.com/Veysel440/go-notes-api/internal/handlers"
	"github.com/Veysel440/go-notes-api/internal/jti"
	"github.com/Veysel440/go-notes-api/internal/logging"
	"github.com/Veysel440/go-notes-api/internal/metrics"
	"github.com/Veysel440/go-notes-api/internal/middleware"
	"github.com/Veysel440/go-notes-api/internal/openapi"
	"github.com/Veysel440/go-notes-api/internal/redisx"
	"github.com/Veysel440/go-notes-api/internal/repos"
	otelsetup "github.com/Veysel440/go-notes-api/internal/trace"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/time/rate"
)

type Server struct {
	cfg  config.Config
	db   *sql.DB
	mx   *metrics.Registry
	log  *slog.Logger
	rdb  *redis.Client
	jtis jti.Store
}

func New(cfg config.Config, db *sql.DB) *Server {
	_, _ = otelsetup.Setup(context.Background(), cfg.OTELEndpoint, cfg.OTELSample, "go-notes-api")
	log := logging.New()
	mx := metrics.New()

	rdb := redisx.New(cfg)
	jtis := jti.Store{RDB: rdb, Prefix: cfg.JTIPrefix}

	return &Server{cfg: cfg, db: db, mx: mx, log: log, rdb: rdb, jtis: jtis}
}

func (s *Server) router() http.Handler {
	r := chi.NewRouter()

	r.Use(
		otelhttp.NewMiddleware("notes-api"),
		middleware.RequestID,
		chimw.RealIP,
		middleware.SecurityHeaders,
		middleware.CORS(s.cfg.CorsOrigins),
		middleware.BodyLimit(s.cfg.MaxBodyBytes),
		middleware.RecoverJSON(s.log),
	)

	if s.rdb != nil {
		r.Use(middleware.
			NewRedisLimiter(s.rdb, s.cfg.RateBurst, time.Minute, middleware.IPKey, s.cfg.RateAllowCIDR).
			Middleware)
	} else {
		r.Use(middleware.NewLimiter(rate.Limit(s.cfg.RateRPS), s.cfg.RateBurst, 5*time.Minute).Middleware)
	}

	r.Use(s.mx.MW, middleware.Logger(s.log), middleware.Audit{DB: s.db}.Middleware)

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

	roles := &repos.Roles{DB: s.db}

	amx := repos.NewAuthMetrics(s.mx.Reg())
	emailLimiter := repos.NewEmailLimiter()
	au := handlers.Auth{
		Cfg:          s.cfg,
		Users:        &repos.Users{DB: s.db},
		Tokens:       &repos.RefreshTokens{DB: s.db},
		Roles:        roles,
		EmailLimiter: emailLimiter,
		Metrics:      amx,
		JTIStore:     s.jtis,
	}
	r.Route("/auth", func(ar chi.Router) {
		if s.rdb != nil {
			ar.Use(middleware.
				NewRedisLimiter(s.rdb, s.cfg.RateAuthBurst, time.Minute,
					func(r *http.Request) string { return "auth:" + middleware.IPKey(r) },
					s.cfg.RateAllowCIDR).
				Middleware)
		} else {
			ar.Use(middleware.NewLimiter(rate.Limit(s.cfg.RateAuthRPS), s.cfg.RateAuthBurst, time.Minute).Middleware)
		}
		ar.Post("/register", au.Register)
		ar.Post("/login", au.Login)
		ar.Post("/refresh", au.Refresh)
		ar.Post("/logout", au.Logout)
	})

	r.Group(func(ar chi.Router) {
		ar.Use(middleware.AuthWith(s.cfg), middleware.RequireRole(roles, "admin"))

		ar.Get("/admin/ping", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		})

		ua := handlers.AdminUsers{Cfg: s.cfg, Users: &repos.Users{DB: s.db}}
		ar.Get("/admin/users", ua.List)

		aroles := handlers.AdminRoles{Cfg: s.cfg, Roles: roles}
		ar.Post("/admin/users/{id}/roles", aroles.Post)

		aa := handlers.AdminAudit{Cfg: s.cfg, Audit: &repos.Audit{DB: s.db}}
		ar.Get("/admin/audit", aa.List)

		aj := handlers.AdminJTI{Store: s.jtis}
		ar.Post("/admin/jti/revoke", aj.Revoke)
	})

	nt := handlers.Notes{Repo: &repos.Notes{DB: s.db, Mx: s.mx}}
	r.Route("/notes", func(pr chi.Router) {
		pr.Use(middleware.AuthWith(s.cfg), middleware.RequireRole(roles, "user"))
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
