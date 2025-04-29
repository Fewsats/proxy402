# LinkshrinkDB justfile
# Usage: just <recipe>

# Default recipe when just is called without arguments
default:
    @just --list

# Run only the PostgreSQL database from docker-compose
db-up:
    docker compose up -d db

# Stop the database container
db-down:
    docker compose down db

# Compile the Go binary
build:
    go build -o linkshrink ./cmd/server

# Run the compiled binary locally with the database running in Docker
run: build
    ./linkshrink

# Run the complete application stack in Docker
docker-up:
    docker compose up -d

# Stop the complete application stack
docker-down:
    docker compose down

# Clean up binaries
clean:
    rm -f linkshrink

# Show container status
status:
    docker compose ps 