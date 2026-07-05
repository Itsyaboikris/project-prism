package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
	"project-prism/server/internal/auth"
	"project-prism/server/internal/config"
	"project-prism/server/internal/db"
	"project-prism/server/internal/mailer"
	"project-prism/server/internal/router"
	"project-prism/server/internal/store"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading environment directly")
	}

	cfg := config.Load()

	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()
	log.Println("database connection established")

	userStore := store.NewUserStore(pool)
	refreshTokenStore := store.NewRefreshTokenStore(pool)
	invitationTokenStore := store.NewInvitationTokenStore(pool)
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
	if user, bootstrapped, err := authService.EnsureBootstrapAdmin(ctx, cfg.BootstrapAdminEmail, cfg.BootstrapAdminPassword); err != nil {
		log.Fatalf("bootstrap admin failed: %v", err)
	} else if bootstrapped {
		log.Printf("bootstrap admin ensured for %s", user.Email)
	}

	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           router.New(pool, cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("API server listening on http://localhost%s", cfg.Address())

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
