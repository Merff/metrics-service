package config

import "os"

const (
	defaultDatabaseURL     = "postgres://postgres:postgres@localhost:5433/metrics?sslmode=disable"
	defaultTestDatabaseURL = "postgres://postgres:postgres@localhost:5433/metrics_test?sslmode=disable"
	defaultHTTPAddr        = ":8080"
)

type Config struct {
	DatabaseURL     string
	TestDatabaseURL string
	HTTPAddr        string
}

func Load() Config {
	return Config{
		DatabaseURL:     getEnv("DATABASE_URL", defaultDatabaseURL),
		TestDatabaseURL: getEnv("TEST_DATABASE_URL", defaultTestDatabaseURL),
		HTTPAddr:        getEnv("HTTP_ADDR", defaultHTTPAddr),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
