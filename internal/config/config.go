package config

import "os"

type Config struct {
	DatabaseURL string
	Port        string
	RabbitMQURL string
}

func Load() Config {
	return Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/price_db?sslmode=disable"),
		Port:        getEnv("PORT", "8080"),
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
