DATABASE_URL ?= postgres://postgres:postgres@localhost:5433/metrics?sslmode=disable
TEST_DATABASE_URL ?= postgres://postgres:postgres@localhost:5433/metrics_test?sslmode=disable
MIGRATE = go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1 -path migrations -database "$(DATABASE_URL)"

.PHONY: migrate-up migrate-down test lint run

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down 1

test:
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test ./...

lint:
	golangci-lint run

run:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/server
