package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port        string
	DatabaseURL string
	OpenAIKey   string
}

func Load() *Config {
	dsn := getEnv("DATABASE_URL", fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", getEnv("DB_USER", "fhir"), getEnv("DB_PASSWORD", "fhir"), getEnv("DB_HOST", "localhost"), getEnv("DB_PORT", "5432"), getEnv("DB_NAME", "fhir_goals")))

	// Add connect_timeout for Railway: fail fast (10s) instead of hanging ~2min on sleeping DB
	if !strings.Contains(dsn, "connect_timeout") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn = dsn + sep + "connect_timeout=10"
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: dsn,
		OpenAIKey:   getEnv("OPENAI_API_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
