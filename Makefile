.PHONY: run build test seed docker-up docker-down migrate

run:
	go run ./cmd/server

build:
	go build -o bin/fhir-goals-engine ./cmd/server

test:
	go test ./... -v -count=1

seed:
	go run seed.go

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

migrate:
	psql "$(DATABASE_URL)" -f migrations/001_create_tables.up.sql

migrate-down:
	psql "$(DATABASE_URL)" -f migrations/001_create_tables.down.sql

lint:
	golangci-lint run ./...
