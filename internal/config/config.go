package config

import "os"

type Config struct {
	DatabaseURL string
	Port        string
}

func Load() Config {
	return Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/price_user_db?sslmode=disable"),
		Port:        getEnv("PORT", "8082"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
