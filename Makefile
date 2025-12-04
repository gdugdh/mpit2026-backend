.PHONY: help run build test docker-up docker-down migrate-up migrate-down migrate-create seed clean

help:
	@echo "Available commands:"
	@echo "  make run          - Run the application"
	@echo "  make build        - Build the application"
	@echo "  make test         - Run tests"
	@echo "  make docker-up    - Start Docker containers"
	@echo "  make docker-down  - Stop Docker containers"
	@echo "  make migrate-up   - Run database migrations"
	@echo "  make migrate-down - Rollback database migrations"
	@echo "  make seed         - Seed database with test data"
	@echo "  make clean        - Clean build artifacts"

run:
	@echo "Running application..."
	go run cmd/api/main.go

build:
	@echo "Building application..."
	go build -o bin/api cmd/api/main.go

test:
	@echo "Running tests..."
	go test -v ./...

docker-up:
	@echo "Starting Docker containers..."
	docker-compose up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5

docker-down:
	@echo "Stopping Docker containers..."
	docker-compose down

migrate-up: docker-up
	@echo "Applying migrations via Docker..."
	docker compose run --rm migrate -path=/migrations -database "postgresql://dating_user:dating_pass@postgres:5432/dating_db?sslmode=disable" up

migrate-down:
	@echo "Rolling back last migration via Docker..."
	docker compose run --rm migrate -path=/migrations -database "postgresql://dating_user:dating_pass@postgres:5432/dating_db?sslmode=disable" down 1

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

seed:
	@echo "Seeding database..."
	@if [ -f scripts/seed.sql ]; then \
		psql -h localhost -U dating_user -d dating_db -f scripts/seed.sql; \
	else \
		echo "No seed file found at scripts/seed.sql"; \
	fi

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf uploads/
	go clean

setup: docker-up migrate-up
	@echo "Setup complete! Copy .env.example to .env and configure it"
	@echo "Then run: make run"

dev: docker-up migrate-up run

.env:
	cp .env.example .env
	@echo "Created .env file. Please configure it."
