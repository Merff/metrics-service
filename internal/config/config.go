package config

import "os"

const (
	defaultDatabaseURL = "postgres://postgres:postgres@localhost:5433/metrics?sslmode=disable"
	defaultHTTPAddr    = ":8080"
)

type Config struct {
	DatabaseURL string
	HTTPAddr    string
}

func Load() Config {
	return Config{
		DatabaseURL: getEnv("DATABASE_URL", defaultDatabaseURL),
		HTTPAddr:    getEnv("HTTP_ADDR", defaultHTTPAddr),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
