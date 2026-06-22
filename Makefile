DB_URL=postgres://postgres:postgres@localhost:5433/metrics?sslmode=disable

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1

test:
	go test ./...

lint:
	golangci-lint run

run:
	go run ./cmd/server
