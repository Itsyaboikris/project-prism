package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"project-prism/server/internal/handlers"
)

func New(pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", handlers.Health)

	r.Route("/api/v1", func(r chi.Router) {
		// application and experiment routes will be added here
		_ = pool
	})

	return r
}
