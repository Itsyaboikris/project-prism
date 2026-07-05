package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"project-prism/server/internal/models"
	"project-prism/server/internal/respond"
)

type accessAuthenticator interface {
	AuthenticateAccessToken(ctx context.Context, token string) (*models.User, error)
}

type Middleware struct {
	auth accessAuthenticator
}

func NewMiddleware(auth accessAuthenticator) *Middleware {
	return &Middleware{auth: auth}
}

func (m *Middleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerTokenFromRequest(r)
		if token == "" {
			respond.Error(w, http.StatusUnauthorized, "missing access token")
			return
		}

		user, err := m.auth.AuthenticateAccessToken(r.Context(), token)
		switch {
		case errors.Is(err, ErrInvalidAccessToken):
			respond.Error(w, http.StatusUnauthorized, "invalid access token")
			return
		case errors.Is(err, ErrInactiveUser):
			respond.Error(w, http.StatusForbidden, "user is inactive")
			return
		case errors.Is(err, ErrForbiddenUser):
			respond.Error(w, http.StatusForbidden, "user is not authorized")
			return
		case err != nil:
			respond.Error(w, http.StatusInternalServerError, "failed to verify access token")
			return
		}

		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
	})
}

func bearerTokenFromRequest(r *http.Request) string {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if token, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
		return strings.TrimSpace(token)
	}

	return ""
}
