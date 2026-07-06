package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"project-prism/server/internal/apiauth"
	"project-prism/server/internal/auth"
	"project-prism/server/internal/config"
	"project-prism/server/internal/handlers"
	"project-prism/server/internal/mailer"
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
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", handlers.Health)

	appStore := store.NewApplicationStore(pool)
	expStore := store.NewExperimentStore(pool)
	branchStore := store.NewBranchStore(pool)
	assignStore := store.NewAssignmentStore(pool)
	eventStore := store.NewEventStore(pool)
	trackedEventStore := store.NewTrackedEventStore(pool)
	userStore := store.NewUserStore(pool)
	refreshTokenStore := store.NewRefreshTokenStore(pool)
	invitationTokenStore := store.NewInvitationTokenStore(pool)
	apiKeyAuth := apiauth.NewMiddleware(appStore)
	smtpMailer := mailer.NewSMTPMailer(mailer.Config{
		Host:      cfg.SMTPHost,
		Port:      cfg.SMTPPort,
		Username:  cfg.SMTPUsername,
		Password:  cfg.SMTPPassword,
		FromEmail: cfg.SMTPFromEmail,
		FromName:  cfg.SMTPFromName,
	})
	authService := auth.NewService(userStore, refreshTokenStore, invitationTokenStore, auth.Config{
		AccessTokenTTL:        cfg.AccessTokenTTL,
		RefreshTokenTTL:       cfg.RefreshTokenTTL,
		InviteTokenTTL:        cfg.InviteTokenTTL,
		RefreshCookieName:     cfg.RefreshCookieName,
		RefreshCookieSecure:   cfg.RefreshCookieSecure,
		RefreshCookieSameSite: cfg.RefreshCookieSameSite,
		RefreshCookiePath:     cfg.RefreshCookiePath,
		RefreshCookieDomain:   cfg.RefreshCookieDomain,
		AppBaseURL:            cfg.AppBaseURL,
	}, cfg.JWTSecret, smtpMailer)
	adminAuth := auth.NewMiddleware(authService)

	appHandler := handlers.NewApplicationHandler(appStore)
	expHandler := handlers.NewExperimentHandler(expStore, branchStore)
	branchHandler := handlers.NewBranchHandler(branchStore, expStore)
	assignHandler := handlers.NewAssignmentHandler(assignStore, eventStore)
	eventHandler := handlers.NewEventHandler(eventStore)
	trackedEventHandler := handlers.NewTrackedEventHandler(trackedEventStore, expStore)
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(authService)

	r.Route("/api/v1", func(r chi.Router) {
		r.With(apiKeyAuth.RequireAPIKey).Post("/assign", assignHandler.Create)
		r.With(apiKeyAuth.RequireAPIKey).Post("/events", eventHandler.Create)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
			r.Get("/invitations/{token}", authHandler.GetInvitation)
			r.Post("/invitations/activate", authHandler.ActivateInvitation)
			r.With(adminAuth.RequireAdmin).Get("/me", authHandler.Me)
		})

		r.With(adminAuth.RequireAdmin).Group(func(r chi.Router) {
			r.Route("/users", func(r chi.Router) {
				r.Get("/", userHandler.List)
				r.Post("/", userHandler.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Patch("/", userHandler.UpdateStatus)
				})
			})

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
					r.Get("/assignments", assignHandler.ListByExperiment)
					r.Get("/events", eventHandler.ListByExperiment)
					r.Get("/tracked-events", trackedEventHandler.List)
					r.Post("/tracked-events", trackedEventHandler.Create)
					r.Put("/tracked-events/{trackedEventID}", trackedEventHandler.Update)
					r.Delete("/tracked-events/{trackedEventID}", trackedEventHandler.Delete)
					r.Get("/dashboard", assignHandler.GetExperimentDashboard)
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
	})

	return r
}
