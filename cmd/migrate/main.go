// Standalone migration runner: applies embedded migrations and exits.
// Usage: DATABASE_URL=... go run ./cmd/migrate
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"casemind/internal/platform/database"
	"casemind/internal/platform/logger"
	"casemind/migrations"
)

func main() {
	log := logger.New(os.Getenv("APP_ENV"))
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := database.Connect(ctx, url)
	if err != nil {
		log.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	n, err := database.Migrate(ctx, pool, migrations.FS, log)
	if err != nil {
		log.Error("migrations failed", "error", err)
		os.Exit(1)
	}
	log.Info("migrations complete", "applied", n)
}
