package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"project-prism/server/internal/apiauth"
	"project-prism/server/internal/config"
	"project-prism/server/internal/handlers"
	"project-prism/server/internal/store"
)

func New(pool *pgxpool.Pool, cfg config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key", "X-Request-Id"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", handlers.Health)

	appStore := store.NewApplicationStore(pool)
	expStore := store.NewExperimentStore(pool)
	branchStore := store.NewBranchStore(pool)
	assignStore := store.NewAssignmentStore(pool)
	apiKeyAuth := apiauth.NewMiddleware(appStore)

	appHandler := handlers.NewApplicationHandler(appStore)
	expHandler := handlers.NewExperimentHandler(expStore, branchStore)
	branchHandler := handlers.NewBranchHandler(branchStore, expStore)
	assignHandler := handlers.NewAssignmentHandler(assignStore)

	r.Route("/api/v1", func(r chi.Router) {
		r.With(apiKeyAuth.RequireAPIKey).Post("/assign", assignHandler.Create)

		r.Route("/applications", func(r chi.Router) {
			r.Get("/", appHandler.List)
			r.Post("/", appHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", appHandler.GetByID)
				r.Put("/", appHandler.Update)
				r.Delete("/", appHandler.Delete)
			})
		})

		r.Route("/applications/{appID}/experiments", func(r chi.Router) {
			r.Get("/", expHandler.List)
			r.Post("/", expHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", expHandler.GetByID)
				r.Put("/", expHandler.Update)
				r.Delete("/", expHandler.Delete)
			})
		})

		r.Route("/applications/{appID}/experiments/{experimentID}/branches", func(r chi.Router) {
			r.Post("/", branchHandler.Create)
			r.Put("/", branchHandler.SaveAll)
			r.Route("/{id}", func(r chi.Router) {
				r.Put("/", branchHandler.Update)
				r.Delete("/", branchHandler.Delete)
			})
		})
	})

	return r
}
