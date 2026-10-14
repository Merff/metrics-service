DATABASE_URL ?= postgres://postgres:postgres@localhost:5433/metrics?sslmode=disable
MIGRATE = go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1 -path migrations -database "$(DATABASE_URL)"
LINT = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2

.PHONY: migrate-up migrate-down test lint run

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down 1

test:
	go test ./...

lint:
	$(LINT) run

run:
	go run ./cmd/server
