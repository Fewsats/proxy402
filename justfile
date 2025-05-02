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

# Clean up binaries
clean:
    rm -f linkshrink

# Show container status
status:
    docker compose ps 

# ===================
# DATABASE MIGRATIONS
# ===================
migrate-up:
    migrate -path store/sqlc/migrations -database "postgres://user:password@localhost:5432/linkshrink?sslmode=disable" -verbose up

migrate-down:
    migrate -path store/sqlc/migrations -database "postgres://user:password@localhost:5432/linkshrink?sslmode=disable" -verbose down 1

migrate-create name:
    migrate create -dir store/sqlc/migrations -seq -ext sql {{name}}

# ===============
# CODE GENERATION 
# ===============
gen: sqlc

sqlc:
    @echo "Generating sql models and queries in Go"
    ./scripts/gen_sqlc_docker.sh

sqlc-check: sqlc
    @echo "Verifying sql code generation."
    @if test -n "$$(git status --porcelain '*.go')"; then \
        echo "SQL models not properly generated! Modified changes:"; \
        git status --porcelain '*.go'; \
        exit 1; \
    else \
        echo "SQL models generated correctly."; \
    fi 