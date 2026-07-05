package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                   string
	DatabaseURL            string
	CORSAllowedOrigins     []string
	JWTSecret              string
	AccessTokenTTL         time.Duration
	RefreshTokenTTL        time.Duration
	InviteTokenTTL         time.Duration
	RefreshCookieName      string
	RefreshCookieSecure    bool
	RefreshCookieSameSite  string
	RefreshCookiePath      string
	RefreshCookieDomain    string
	AppBaseURL             string
	SMTPHost               string
	SMTPPort               int
	SMTPUsername           string
	SMTPPassword           string
	SMTPFromEmail          string
	SMTPFromName           string
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	origins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if origins == "" {
		origins = "http://localhost:5173,http://127.0.0.1:5500"
	}

	jwtSecret := os.Getenv("AUTH_JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-me"
	}

	return Config{
		Port:                   port,
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		CORSAllowedOrigins:     splitTrimmed(origins),
		JWTSecret:              jwtSecret,
		AccessTokenTTL:         parseDuration("AUTH_ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:        parseDuration("AUTH_REFRESH_TOKEN_TTL", 7*24*time.Hour),
		InviteTokenTTL:         parseDuration("AUTH_INVITE_TOKEN_TTL", 72*time.Hour),
		RefreshCookieName:      envOrDefault("AUTH_REFRESH_COOKIE_NAME", "prism_refresh"),
		RefreshCookieSecure:    parseBool("AUTH_COOKIE_SECURE", false),
		RefreshCookieSameSite:  envOrDefault("AUTH_COOKIE_SAME_SITE", "lax"),
		RefreshCookiePath:      envOrDefault("AUTH_REFRESH_COOKIE_PATH", "/api/v1/auth"),
		RefreshCookieDomain:    strings.TrimSpace(os.Getenv("AUTH_COOKIE_DOMAIN")),
		AppBaseURL:             strings.TrimSpace(os.Getenv("APP_BASE_URL")),
		SMTPHost:               strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:               parseInt("SMTP_PORT", 587),
		SMTPUsername:           strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:           strings.TrimSpace(os.Getenv("SMTP_PASSWORD")),
		SMTPFromEmail:          strings.TrimSpace(os.Getenv("SMTP_FROM_EMAIL")),
		SMTPFromName:           strings.TrimSpace(os.Getenv("SMTP_FROM_NAME")),
		BootstrapAdminEmail:    strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL")),
		BootstrapAdminPassword: strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")),
	}
}

func splitTrimmed(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func (c Config) Address() string {
	return ":" + c.Port
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}

	return fallback
}

func parseDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("invalid %s value %q, using default %s", key, raw, fallback)
		return fallback
	}

	return value
}

func parseBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		log.Printf("invalid %s value %q, using default %t", key, raw, fallback)
		return fallback
	}

	return value
}

func parseInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("invalid %s value %q, using default %d", key, raw, fallback)
		return fallback
	}

	return value
}
