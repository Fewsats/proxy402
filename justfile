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

logs:
    journalctl --user -u linkshrink --no-pager

status:
    systemctl --user status linkshrink

restart:
    systemctl --user start linkshrink

restart:
    systemctl --user restart linkshrink

# Clean up binaries
clean:
    rm -f linkshrink

# Show container status
status:
    docker compose ps 