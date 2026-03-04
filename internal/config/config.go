package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	OpenAIKey   string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", getEnv("DB_USER", "fhir"), getEnv("DB_PASSWORD", "fhir"), getEnv("DB_HOST", "localhost"), getEnv("DB_PORT", "5432"), getEnv("DB_NAME", "fhir_goals"))),
		OpenAIKey:   getEnv("OPENAI_API_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
