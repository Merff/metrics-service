package config

import "os"

const defaultDatabaseURL = "postgres://postgres:postgres@localhost:5433/metrics?sslmode=disable"

type Config struct {
	DatabaseURL string
}

func Load() Config {
	return Config{
		DatabaseURL: getEnv("DATABASE_URL", defaultDatabaseURL),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
