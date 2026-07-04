package config

import (
	"os"
	"strings"
)

type Config struct {
	Port               string
	DatabaseURL        string
	CORSAllowedOrigins []string
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

	return Config{
		Port:               port,
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		CORSAllowedOrigins: splitTrimmed(origins),
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
