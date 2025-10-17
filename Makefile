# TimeTask Makefile

.PHONY: build clean server client test deps help docker-up docker-down

# Default target
help:
	@echo "TimeTask - Available commands:"
	@echo "  make docker-up   - Start with Docker (recommended)"
	@echo "  make docker-down - Stop Docker containers"
	@echo "  make build       - Build both server and client"
	@echo "  make server      - Build and run server locally"
	@echo "  make client      - Build and run client"
	@echo "  make test        - Run tests"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make dev-setup   - Full local development setup"

# Install dependencies
deps:
	go mod tidy
	go mod download

# Docker commands (recommended)
docker-up:
	@echo "Starting TimeTask with Docker..."
	docker-compose up -d
	@echo "Server running! Now run 'make client' to connect."

docker-down:
	@echo "Stopping Docker containers..."
	docker-compose down

# Build both applications
build: deps
	@echo "Building TimeTask applications..."
	go build -o timetask-server ./cmd/server
	go build -o timetask-client ./cmd/client
	@echo "Build complete! Use 'make docker-up' + 'make client' or 'make server' + 'make client'."

# Build and run server locally
server: 
	@echo "Building and starting server locally..."
	go build -o timetask-server ./cmd/server
	./timetask-server

# Build and run client
client:
	@echo "Building and starting client..."
	go build -o timetask-client ./cmd/client
	./timetask-client

# Run tests
test:
	go test ./...
	go test -race ./...

# Setup local development database
setup-db:
	@echo "Setting up local development database..."
	createdb timetask || echo "Database may already exist"
	psql -c "CREATE USER timetask WITH PASSWORD 'timetask';" || echo "User may already exist"
	psql -c "GRANT ALL PRIVILEGES ON DATABASE timetask TO timetask;" || echo "Privileges may already be granted"
	@echo "Database setup complete!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f timetask-server timetask-client debug-client
	go clean
	@echo "Clean complete!"

# Full local development setup
dev-setup: deps setup-db build
	@echo "Local development setup complete!"
	@echo "Run 'make server' in one terminal and 'make client' in another."
	@echo ""
	@echo "💡 Tip: Use 'make docker-up' + 'make client' for easier development!"