package apiauth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"project-prism/server/internal/models"
	"project-prism/server/internal/respond"
	"project-prism/server/internal/store"
)

type applicationLookup interface {
	GetByAPIKey(ctx context.Context, apiKey string) (*models.Application, error)
}

type Middleware struct {
	apps applicationLookup
}

func NewMiddleware(apps applicationLookup) *Middleware {
	return &Middleware{apps: apps}
}

func (m *Middleware) RequireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := strings.TrimSpace(apiKeyFromRequest(r))
		if apiKey == "" {
			respond.Error(w, http.StatusUnauthorized, "missing api key")
			return
		}

		app, err := m.apps.GetByAPIKey(r.Context(), apiKey)
		if errors.Is(err, store.ErrNotFound) {
			respond.Error(w, http.StatusUnauthorized, "invalid api key")
			return
		}
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to verify api key")
			return
		}
		if app.Status != models.ApplicationStatusActive {
			respond.Error(w, http.StatusForbidden, "application is inactive")
			return
		}

		next.ServeHTTP(w, r.WithContext(WithApplication(r.Context(), app)))
	})
}

func apiKeyFromRequest(r *http.Request) string {
	if key := r.Header.Get("X-API-Key"); key != "" {
		return key
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if bearerToken, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
		return strings.TrimSpace(bearerToken)
	}

	return ""
}
