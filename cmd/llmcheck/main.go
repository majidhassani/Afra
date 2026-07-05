// llmcheck verifies LLM connectivity with the same client and configuration
// the API server uses, without starting the server. Useful for diagnosing
// provider issues in deploys:
//
//	set -a; source .env; set +a; go run ./cmd/llmcheck
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"casemind/internal/config"
	"casemind/internal/llm"
	llmmock "casemind/internal/llm/mock"
	"casemind/internal/llm/openaicompat"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Only the LLM section matters here; stub the unrelated required vars so
	// config.Load's full validation doesn't block a connectivity check.
	if os.Getenv("DATABASE_URL") == "" {
		os.Setenv("DATABASE_URL", "postgres://unused/llmcheck")
	}
	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "llmcheck")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Error("config invalid", "error", err)
		os.Exit(1)
	}

	var client llm.Client
	if cfg.LLM.IsMock() {
		client = llmmock.New()
	} else {
		client = openaicompat.New(cfg.LLM, log)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		log.Error("llm check failed", "provider", client.Name(), "error", err)
		os.Exit(1)
	}
	fmt.Printf("OK: %s is reachable and answering\n", client.Name())
}
