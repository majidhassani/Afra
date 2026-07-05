// Package config loads all runtime configuration from environment variables.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// ProviderMock runs the full game loop with canned responses and no
// external model. Any other provider value ("gemini", "openai", "glm", ...)
// uses the OpenAI-compatible chat-completions client and only labels the
// logs/metrics — there is no provider-specific code path.
const ProviderMock = "mock"

type LLM struct {
	Provider    string
	BaseURL     string
	APIKey      string
	Model       string
	Timeout     time.Duration
	MaxTokens   int
	Temperature float64
}

// IsMock reports whether the offline mock provider is selected.
func (l LLM) IsMock() bool { return l.Provider == ProviderMock }

// validate returns a descriptive error for every misconfigured LLM variable.
func (l LLM) validate() error {
	if l.Provider == "" {
		return fmt.Errorf("LLM_PROVIDER is required (e.g. \"gemini\", or \"mock\" for offline development)")
	}
	if l.IsMock() {
		return nil
	}
	if l.BaseURL == "" {
		return fmt.Errorf("LLM_BASE_URL is required when LLM_PROVIDER=%s (OpenAI-compatible /v1 gateway URL)", l.Provider)
	}
	u, err := url.Parse(l.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("LLM_BASE_URL %q is not a valid http(s) URL", l.BaseURL)
	}
	if l.APIKey == "" {
		return fmt.Errorf("LLM_API_KEY is required when LLM_PROVIDER=%s", l.Provider)
	}
	if l.Model == "" {
		return fmt.Errorf("LLM_MODEL is required when LLM_PROVIDER=%s (e.g. \"Gemini-3.1-Flash-Lite-Preview\")", l.Provider)
	}
	if l.Timeout <= 0 {
		return fmt.Errorf("LLM_TIMEOUT_SECONDS must be a positive integer, got %s", l.Timeout)
	}
	if l.MaxTokens <= 0 {
		return fmt.Errorf("LLM_MAX_TOKENS must be a positive integer, got %d", l.MaxTokens)
	}
	if l.Temperature < 0 || l.Temperature > 2 {
		return fmt.Errorf("LLM_TEMPERATURE must be in [0, 2], got %g", l.Temperature)
	}
	return nil
}

// Image configures an optional OpenAI-compatible image-generation API used
// for avatars. When BaseURL is empty, avatar generation falls back to the
// built-in procedural generator (no external calls).
type Image struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
	Timeout  time.Duration
}

// Enabled reports whether an external image API is configured.
func (i Image) Enabled() bool { return i.BaseURL != "" }

func (i Image) validate() error {
	if !i.Enabled() {
		return nil
	}
	u, err := url.Parse(i.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("IMAGE_API_BASE_URL %q is not a valid http(s) URL", i.BaseURL)
	}
	if i.APIKey == "" {
		return fmt.Errorf("IMAGE_API_KEY is required when IMAGE_API_BASE_URL is set")
	}
	if i.Model == "" {
		return fmt.Errorf("IMAGE_API_MODEL is required when IMAGE_API_BASE_URL is set")
	}
	return nil
}

type MinIO struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type RateLimit struct {
	AuthPerMinute  int
	AgentPerMinute int
}

type Config struct {
	Env  string
	Port string

	DatabaseURL string
	AutoMigrate bool

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration

	LLM       LLM
	Image     Image
	MinIO     MinIO
	QdrantURL string
	RateLimit RateLimit
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:  getenv("APP_ENV", "development"),
		Port: getenv("HTTP_PORT", "8080"),

		DatabaseURL: getenv("DATABASE_URL", ""),
		AutoMigrate: getbool("AUTO_MIGRATE", true),

		RedisAddr:     getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getenv("REDIS_PASSWORD", ""),
		RedisDB:       getint("REDIS_DB", 0),

		JWTSecret:  getenv("JWT_SECRET", ""),
		AccessTTL:  getdur("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTTL: getdur("JWT_REFRESH_TTL", 30*24*time.Hour),

		LLM: LLM{
			// Normalized to lowercase so LLM_PROVIDER=Gemini and gemini are
			// the same provider instead of silently selecting the mock.
			Provider:    strings.ToLower(strings.TrimSpace(getenv("LLM_PROVIDER", ProviderMock))),
			BaseURL:     getenv("LLM_BASE_URL", ""),
			APIKey:      getenv("LLM_API_KEY", ""),
			Model:       getenv("LLM_MODEL", ""),
			Timeout:     time.Duration(getint("LLM_TIMEOUT_SECONDS", 60)) * time.Second,
			MaxTokens:   getint("LLM_MAX_TOKENS", 4096),
			Temperature: getfloat("LLM_TEMPERATURE", 0.7),
		},
		Image: Image{
			Provider: strings.ToLower(strings.TrimSpace(getenv("IMAGE_API_PROVIDER", "image-api"))),
			BaseURL:  getenv("IMAGE_API_BASE_URL", ""),
			APIKey:   getenv("IMAGE_API_KEY", ""),
			Model:    getenv("IMAGE_API_MODEL", ""),
			Timeout:  time.Duration(getint("IMAGE_API_TIMEOUT_SECONDS", 120)) * time.Second,
		},
		MinIO: MinIO{
			Endpoint:  getenv("MINIO_ENDPOINT", ""),
			AccessKey: getenv("MINIO_ACCESS_KEY", ""),
			SecretKey: getenv("MINIO_SECRET_KEY", ""),
			Bucket:    getenv("MINIO_BUCKET", "casemind"),
			UseSSL:    getbool("MINIO_USE_SSL", false),
		},
		QdrantURL: getenv("QDRANT_URL", ""),
		RateLimit: RateLimit{
			AuthPerMinute:  getint("RATE_LIMIT_AUTH_PER_MINUTE", 20),
			AgentPerMinute: getint("RATE_LIMIT_AGENT_PER_MINUTE", 30),
		},
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if err := cfg.LLM.validate(); err != nil {
		return nil, err
	}
	if err := cfg.Image.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getint(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getfloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func getbool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getdur(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
