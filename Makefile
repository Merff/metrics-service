DATABASE_URL ?= postgres://postgres:postgres@localhost:5433/metrics?sslmode=disable

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

test:
	DATABASE_URL="$(DATABASE_URL)" go test ./...

lint:
	golangci-lint run

run:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/server
