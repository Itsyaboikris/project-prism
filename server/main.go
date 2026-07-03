package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
	"project-prism/server/internal/config"
	"project-prism/server/internal/db"
	"project-prism/server/internal/router"
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

	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           router.New(pool),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("API server listening on http://localhost%s", cfg.Address())

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
