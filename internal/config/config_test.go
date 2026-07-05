package config

import (
	"strings"
	"testing"
	"time"
)

func validRemote() LLM {
	return LLM{
		Provider:    "gemini",
		BaseURL:     "https://arvancloudai.ir/gateway/models/x/v1",
		APIKey:      "key",
		Model:       "Gemini-3.1-Flash-Lite-Preview",
		Timeout:     60 * time.Second,
		MaxTokens:   4096,
		Temperature: 0.7,
	}
}

func TestLLMValidate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*LLM)
		wantErr string
	}{
		{"valid", func(l *LLM) {}, ""},
		{"mock needs nothing", func(l *LLM) { *l = LLM{Provider: "mock"} }, ""},
		{"empty provider", func(l *LLM) { l.Provider = "" }, "LLM_PROVIDER"},
		{"missing base url", func(l *LLM) { l.BaseURL = "" }, "LLM_BASE_URL"},
		{"bad base url", func(l *LLM) { l.BaseURL = "not a url" }, "LLM_BASE_URL"},
		{"missing key", func(l *LLM) { l.APIKey = "" }, "LLM_API_KEY"},
		{"missing model", func(l *LLM) { l.Model = "" }, "LLM_MODEL"},
		{"zero timeout", func(l *LLM) { l.Timeout = 0 }, "LLM_TIMEOUT_SECONDS"},
		{"zero max tokens", func(l *LLM) { l.MaxTokens = 0 }, "LLM_MAX_TOKENS"},
		{"temperature too high", func(l *LLM) { l.Temperature = 3 }, "LLM_TEMPERATURE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := validRemote()
			tc.mutate(&l)
			err := l.validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error mentioning %s, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestProviderNormalization(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("LLM_PROVIDER", "Gemini") // mixed case must not fall back to mock
	t.Setenv("LLM_BASE_URL", "https://example.com/v1")
	t.Setenv("LLM_API_KEY", "k")
	t.Setenv("LLM_MODEL", "m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LLM.Provider != "gemini" {
		t.Errorf("provider = %q, want normalized \"gemini\"", cfg.LLM.Provider)
	}
	if cfg.LLM.IsMock() {
		t.Error("Gemini must not be treated as mock")
	}
}
