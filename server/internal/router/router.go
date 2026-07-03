package router

import (
	"net/http"

	"project-prism/server/internal/handlers"
)

func New() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handlers.Health)

	return mux
}
