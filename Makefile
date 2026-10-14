DATABASE_URL ?= postgres://postgres:postgres@localhost:5433/metrics?sslmode=disable
TEST_DB_NAME ?= metrics_test
TEST_DATABASE_URL ?= postgres://postgres:postgres@localhost:5433/$(TEST_DB_NAME)?sslmode=disable

.PHONY: migrate-up migrate-down test-db test lint run

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

test-db:
	docker compose exec -T postgres psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '$(TEST_DB_NAME)'" | grep -q 1 \
		|| docker compose exec -T postgres createdb -U postgres $(TEST_DB_NAME)
	migrate -path migrations -database "$(TEST_DATABASE_URL)" up

test: test-db
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test ./...

lint:
	golangci-lint run

run:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/server
