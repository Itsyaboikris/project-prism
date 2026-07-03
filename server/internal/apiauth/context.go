package apiauth

import (
	"context"

	"project-prism/server/internal/models"
)

type contextKey string

const applicationContextKey contextKey = "application"

func WithApplication(ctx context.Context, app *models.Application) context.Context {
	return context.WithValue(ctx, applicationContextKey, app)
}

func ApplicationFromContext(ctx context.Context) (*models.Application, bool) {
	app, ok := ctx.Value(applicationContextKey).(*models.Application)
	return app, ok
}
