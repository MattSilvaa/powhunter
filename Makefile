.PHONY: dev build clean server client

dev:
	@echo "Starting development environment..."
	@make -j2 server client
build:
	@echo "Building client and server..."
	@cd client && deno task build
	@cd server && go build -o bin/powhunter cmd/main.go

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf client/dist
	@rm -rf server/bin

server:
	@echo "Starting server..."
	@cd server && go run cmd/main.go

client:
	@echo "Starting client..."
	@cd client && deno task dev

install:
	@echo "Installing dependencies..."
	@cd server && go mod tidy
	@cd client && deno install

test:
	@echo "Running tests..."
	@cd server && go test ./...
	@cd client && deno test 