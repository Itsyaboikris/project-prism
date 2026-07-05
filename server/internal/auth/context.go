package auth

import (
	"context"

	"project-prism/server/internal/models"
)

type contextKey string

const userContextKey contextKey = "auth.user"

func WithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userContextKey).(*models.User)
	return user, ok
}
