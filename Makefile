.PHONY: all build test lint migrate sqlc clean dev

all: sqlc build

build:
	cd backend && go build -o bin/berth-api ./cmd/api

test:
	cd backend && go test -race -v ./...

lint:
	cd backend && golangci-lint run ./...

migrate-up:
	cd backend && migrate -path migrations -database "postgres://berth:berth@localhost:5432/berth?sslmode=disable" up

migrate-down:
	cd backend && migrate -path migrations -database "postgres://berth:berth@localhost:5432/berth?sslmode=disable" down

sqlc:
	cd backend && sqlc generate

dev:
	cd infra && docker compose up -d
	@echo "Infrastructure started. Run 'make migrate-up' then 'cd backend && go run ./cmd/api'"

clean:
	cd infra && docker compose down -v
	rm -rf backend/bin/
