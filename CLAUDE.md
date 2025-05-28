# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Coding Style

Always try to keep complexity low and make code concise. Follow the style of Jeremy Howard & Fast.ai.
Less code is almost always desirable.
Make sure no code can panic in golang, point out when something can trigger one.
Do not use single-line 'if _, err := statment; err != nil' always split it in 2 lines.

## Project Overview

Proxy402 (previously Linkshrink) is a service that lets you monetize APIs by requiring x402 payments on the Base blockchain before accessing your endpoints. It serves as a proxy that enforces payment requirements for protected routes.

Key components:
- **Backend**: Go server using Gin framework
- **Database**: PostgreSQL with sqlc for type-safe queries
- **Authentication**: OAuth (Google)
- **Payments**: x402 protocol on Base blockchain
- **Storage**: Cloudflare R2 for file storage
- **Client**: TypeScript example client demonstrating payment flows

## Common Commands

### Development

```bash
# Run PostgreSQL database in Docker
just db-up

# Build the Go application
just build

# Run with live reload (requires air)
just dev

# Shut down the database
just db-down

# Show container status
just status

# Clean up binaries
just clean
```

### Database Operations

```bash
# Run migrations
just migrate-up

# Rollback the most recent migration
just migrate-down

# Create a new migration file
just migrate-create <name>

# Generate SQL models and queries in Go
just sqlc

# List database tables
just list-tables

# Show contents of a specific table
just show-table <table_name>
```

### Client Commands

```bash
# Run the client against a specific URL
cd client
# Run client on Base Sepolia testnet
npm run client -- <proxy402_url>
```


## Architecture

The application follows a layered architecture:

1. **HTTP Layer**: Defined in `server/server.go`, handles routing and middleware
2. **Handler Layer**: Contains request/response logic for different domains:
   - `auth/handler.go`: Authentication flow
   - `routes/handler.go`: Managing paid routes
   - `purchases/handler.go`: Purchase-related operations
   - `ui/handler.go`: UI endpoints
3. **Service Layer**: Business logic
   - `routes/service.go`: PaidRouteService
   - `purchases/service.go`: PurchaseService
   - `users/service.go`: UserService
   - `auth/auth.go`: AuthService
4. **Store Layer**: Data persistence
   - `store/store.go`: Database operations
   - `store/sqlc/`: Generated SQL code

### Key Workflows

1. **Route Creation**: Users create monetized routes that proxy to target URLs
2. **Payment Flow**:
   - Client requests protected resource
   - Server returns 402 with x402 payment token
   - Client pays using Base blockchain
   - Client retries with proof of payment
   - Server forwards request to target URL

### Configuration

The application uses environment variables for configuration:
- Database connection details
- OAuth credentials
- Blockchain payment addresses
- App port and other settings

Important settings are defined in the `.env` file (copied from `.env.example`).


## Deployment

The application is automatically deployed with the github action @.github/workflows/deploy.yml

## Development Philosophy

This project follows an **iterative simplification approach** that can be applied to any codebase or technology:

**Core Process**: Question every requirement, remove rather than add, ensure deep understanding before adding complexity. The goal is not just simple code, but comprehensive understanding of what you're building.

**Universal Principles**:
- **Understand first, optimize second** - Don't proceed until current concepts are clear
- **Question everything** - "Is this actually needed? What's the simplest way?"
- **Delete over create** - Prefer removing code/features over adding new ones
- **One concept at a time** - Master each piece before introducing more
- **Make it work, then make it simple** - Get basic functionality, then ruthlessly simplify
- **Learn through iteration** - Each simplification should deepen understanding

**Application**: Start with working but complex code. Ask "Why does this exist?" For each component, try removing it or simplifying it. If removal breaks functionality, understand why, then find the minimal implementation that preserves the essential behavior.

**Result**: Simpler code AND deeper understanding. By questioning assumptions and preferring minimal solutions, you build better software while actually learning the domain.