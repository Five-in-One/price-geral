package config

import "os"

type Config struct {
	DatabaseURL string
	Port        string
	// JWTSecret must match price-auth's, so its tokens can be verified here.
	JWTSecret string
}

func Load() Config {
	return Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/price_user_db?sslmode=disable"),
		Port:        getEnv("PORT", "8082"),
		JWTSecret:   getEnv("JWT_SECRET", "dev-secret-change-me"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
