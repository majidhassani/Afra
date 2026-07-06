// Command llmsmoke is a live provider smoke test: it pings the configured LLM
// provider and runs a small bilingual guidance-style completion, checking that
// a Persian request comes back in Persian script and an English one in Latin
// script. It exits non-zero on any failure so it can gate a release.
//
// Usage: LLM_PROVIDER=gemini LLM_BASE_URL=... LLM_API_KEY=... LLM_MODEL=... go run ./cmd/llmsmoke
package main

import (
	"context"
	"fmt"
	"os"
	"time"
	"unicode"

	"casemind/internal/agent/runtime"
	"casemind/internal/config"
	"casemind/internal/llm"
	llmmock "casemind/internal/llm/mock"
	"casemind/internal/llm/openaicompat"
	"casemind/internal/platform/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "llmsmoke: FAIL:", err)
		os.Exit(1)
	}
	fmt.Println("llmsmoke: PASS")
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		// Smoke test does not need DB/JWT; tolerate their absence.
		if cfg == nil {
			cfg = &config.Config{Env: "development"}
			cfg.LLM.Provider = os.Getenv("LLM_PROVIDER")
		}
	}
	log := logger.New("development")

	var client llm.Client
	if cfg.LLM.Provider == "" || cfg.LLM.IsMock() {
		client = llmmock.New()
	} else {
		client = openaicompat.New(cfg.LLM, log)
	}
	fmt.Println("llmsmoke: provider =", client.Name())

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	fmt.Println("llmsmoke: ping ok")

	// The mock provider only answers known agent task types, not free-form
	// prompts — language checks require a real provider.
	if client.Name() == "mock" {
		fmt.Println("llmsmoke: mock provider — skipping language checks (set a real LLM_PROVIDER to test them)")
		return nil
	}

	// Persian request must come back in Persian; English in English.
	if err := checkLanguage(ctx, client, "fa", isPersian); err != nil {
		return err
	}
	if err := checkLanguage(ctx, client, "en", isLatin); err != nil {
		return err
	}
	return nil
}

func checkLanguage(ctx context.Context, client llm.Client, lang string, ok func(string) bool) error {
	resp, err := client.Chat(ctx, llm.Request{
		Temperature: 0.3,
		MaxTokens:   200,
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: runtime.LanguageDirective(lang) +
				"\nYou are Mission Control. Answer in one short sentence."},
			{Role: llm.RoleUser, Content: "Give the agent a brief encouragement to keep investigating."},
		},
	})
	if err != nil {
		return fmt.Errorf("%s chat: %w", lang, err)
	}
	fmt.Printf("llmsmoke: %s -> %q\n", lang, resp.Content)
	if !ok(resp.Content) {
		return fmt.Errorf("%s response not in the expected script: %q", lang, resp.Content)
	}
	return nil
}

// isPersian reports whether the text contains Arabic-script (Persian) letters.
func isPersian(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Arabic, r) {
			return true
		}
	}
	return false
}

// isLatin reports whether the text is predominantly Latin letters (no Persian).
func isLatin(s string) bool {
	latin, arabic := 0, 0
	for _, r := range s {
		switch {
		case unicode.Is(unicode.Latin, r):
			latin++
		case unicode.Is(unicode.Arabic, r):
			arabic++
		}
	}
	return latin > 0 && arabic == 0
}
