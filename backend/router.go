package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server holds the dependencies shared by all handlers.
type Server struct {
	cfg   Config
	db    *pgxpool.Pool
	repo  *Repository
	leads *LeadService
	auth  *Auth
}

// NewApp wires config, database and routes. It is used by both the local
// server (cmd/server) and the Vercel function (api/index.go).
func NewApp(ctx context.Context) (http.Handler, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	db, err := NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	repo := NewRepository(db)
	s := &Server{
		cfg:   cfg,
		db:    db,
		repo:  repo,
		leads: NewLeadService(repo),
		auth:  NewAuth(cfg),
	}
	return s.routes(), nil
}

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	r.Route("/api", func(r chi.Router) {
		// Public
		r.Get("/health", s.handleHealth)
		r.Post("/leads", s.handleCreateLead)

		r.With(requireJSON).Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)

		// Admin only
		r.Group(func(r chi.Router) {
			r.Use(s.auth.requireAuth)
			r.Use(requireJSON)

			r.Get("/auth/me", s.handleMe)
			r.Get("/leads", s.handleListLeads)
			r.Get("/leads/export", s.handleExport)
			r.Get("/leads/stats", s.handleCampaignStats)
			r.Get("/leads/timeseries", s.handleWeeklyLeads)
			r.Get("/leads/{id}", s.handleGetLead)
			r.Patch("/leads/{id}/status", s.handleUpdateStatus)
			r.Delete("/leads/{id}", s.handleDeleteLead)

			r.Get("/campaigns", s.handleListCampaigns)
			r.Post("/campaigns", s.handleCreateCampaign)
			r.Put("/campaigns/{id}", s.handleUpdateCampaign)
			r.Delete("/campaigns/{id}", s.handleDeleteCampaign)
		})
	})
	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "database": "unreachable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "database": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
