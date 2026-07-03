package main

import (
	"log"
	"net/http"
	"time"

	"project-prism/server/internal/config"
	"project-prism/server/internal/router"
)

func main() {
	cfg := config.Load()

	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           router.New(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("API server listening on http://localhost%s", cfg.Address())

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
