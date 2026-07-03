// Package config loads all runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type LLM struct {
	Provider    string // "glm" or "mock"
	BaseURL     string
	APIKey      string
	Model       string
	Timeout     time.Duration
	MaxTokens   int
	Temperature float64
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
			Provider:    getenv("LLM_PROVIDER", "mock"),
			BaseURL:     getenv("LLM_BASE_URL", ""),
			APIKey:      getenv("LLM_API_KEY", ""),
			Model:       getenv("LLM_MODEL", "glm-5.2"),
			Timeout:     time.Duration(getint("LLM_TIMEOUT_SECONDS", 60)) * time.Second,
			MaxTokens:   getint("LLM_MAX_TOKENS", 4096),
			Temperature: getfloat("LLM_TEMPERATURE", 0.7),
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
	if cfg.LLM.Provider == "glm" && cfg.LLM.BaseURL == "" {
		return nil, fmt.Errorf("LLM_BASE_URL is required when LLM_PROVIDER=glm")
	}
	if cfg.LLM.Provider == "glm" && cfg.LLM.APIKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY is required when LLM_PROVIDER=glm")
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
