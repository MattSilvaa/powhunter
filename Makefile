.PHONY: dev dev-caddy build clean server client caddy db-setup db-migrate db-reset generate-db-code db-seed check-forecasts

dev:
	@echo "Starting development environment..."
	@make -j2 server client

dev-caddy:
	@echo "Starting development environment with Caddy..."
	@echo "Caddy will serve the app on http://localhost:3000"
	@make -j2 server caddy
build:
	@echo "Building client and server..."
	@cd client && bun run build
	@cd server && go build -o bin/powhunter cmd/api/main.go
	@cd server && go build -o bin/check_forecasts cmd/forecaster/main.go

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf client/dist
	@rm -rf server/bin

server:
	@echo "Starting server..."
	@cd server && go run cmd/api/main.go

client:
	@echo "Starting client..."
	@cd client && bun run dev

caddy:
	@echo "Starting Caddy reverse proxy..."
	@caddy run --config Caddyfile --adapter caddyfile

start-prod:
	@echo "Starting production environment..."
	@make -j2 prod-server prod-client

start-caddy:
	@echo "Starting production environment with Caddy..."
	@echo "Make sure to build first: make build"
	@make -j2 prod-server caddy

prod-server:
	@cd server && ./bin/powhunter

prod-client:
	@cd client && bun run start

start-forecaster:
	@echo "Starting forecaster..."
	@cd server && go run cmd/forecaster/main.go


install:
	@echo "Installing dependencies..."
	@cd server && go mod tidy
	@cd client && bun install
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install github.com/pressly/goose/v3/cmd/goose@latest

test:
	@echo "Running tests..."
	@cd server && go test ./...
	@cd client && bun test

# Database commands
db-setup:
	@cd server && docker compose up -d postgres
	sleep 2
	./server/scripts/create_db.sh

db-migrate:
	@echo "Running database migrations..."
	@cd server && ./scripts/migrate_db.sh

db-drop:
	@echo "Dropping database..."
	@cd server && ./scripts/drop_db.sh

generate-db-code:
	@echo "Generating database code..."
	@cd server && sqlc generate

db-seed:
	@echo "Seeding database with initial data..."
	@cd server && go run cmd/seed/main.go