.PHONY: dev build clean server client db-setup db-migrate db-reset generate-db-code db-seed check-forecasts

dev:
	@echo "Starting development environment..."
	@make -j2 server client
build:
	@echo "Building client and server..."
	@cd client && deno task build
	@cd server && go build -o bin/powhunter cmd/main.go
	@cd server && go build -o bin/check_forecasts cmd/check_forecasts/main.go

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf client/dist
	@rm -rf server/bin

server:
	@echo "Starting server..."
	@cd server && go run cmd/api/main.go

client:
	@echo "Starting client..."
	@cd client && deno task dev

install:
	@echo "Installing dependencies..."
	@cd server && go mod tidy
	@cd client && deno install
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install github.com/pressly/goose/v3/cmd/goose@latest

test:
	@echo "Running tests..."
	@cd server && go test ./...
	@cd client && deno test

# Database commands
db-setup:
	@cd server && docker compose up -d postgres
	sleep 2
	./server/scripts/create_db.sh

db-migrate:
	@echo "Running database migrations..."
	@cd server && goose -dir internal/db/migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=powhunter sslmode=disable" up

db-reset:
	@echo "Resetting database..."
	@dropdb -U postgres powhunter || echo "Database may not exist"
	@make db-setup

generate-db-code:
	@echo "Generating database code..."
	@cd server && sqlc generate

db-seed:
	@echo "Seeding database with initial data..."
	@cd server && go run cmd/seed/main.go

check-forecasts:
	@echo "Checking snow forecasts..."
	@cd server && go run cmd/check_forecasts/main.go

check-forecasts-send-sms:
	@echo "Checking snow forecasts and sending SMS..."
	@cd server && go run cmd/check_forecasts/main.go -send-sms