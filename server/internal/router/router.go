package router

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"project-prism/server/internal/handlers"
)

func New(pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handlers.Health)

	_ = pool // pool will be wired into handlers as routes are added

	return mux
}
