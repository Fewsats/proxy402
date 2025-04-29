# LinkShrink - Paid API Proxy Service

LinkShrink acts as a configurable, paid proxy for other API endpoints. Registered users can define routes consisting of a target URL, HTTP method, and a price. Accessing the generated short link for a route requires an L402 payment (verification currently pending implementation) before the request is proxied to the target URL.

The service uses Go, Gin, PostgreSQL (with GORM), and JWT for authentication.

## Features

*   User registration (`POST /register`) and login (`POST /login`).
*   JWT-based authentication for protected endpoints.
*   **Paid Route Creation:** Define target endpoints with associated methods and prices (`POST /links/shrink`).
*   **Proxying:** Requests to `/<shortCode>` are proxied to the configured target URL and method.
*   **Payment Enforcement (TODO):** Intended to verify L402 payments via a facilitator before proxying.
*   **Payment Count Tracking:** Tracks the number of successful accesses/payments for each route.
*   **Route Management:** List (`GET /links`) and delete (`DELETE /links/:linkID`) configured routes.

## Tech Stack

*   **Language:** Go (1.23.3)
*   **Web Framework:** Gin Gonic
*   **Database:** PostgreSQL
*   **ORM:** GORM
*   **Authentication:** JWT (golang-jwt/jwt)
*   **Password Hashing:** bcrypt
*   **Payment Protocol (Partial):** L402 (using vendored code from `github.com/coinbase/x402`)
*   **Containerization:** Docker & Docker Compose

## Local Setup & Running (Docker)

This project uses Docker Compose to simplify local development and testing.

**Prerequisites:**

*   Docker ([Install Docker](https://docs.docker.com/get-docker/))
*   Docker Compose ([Included with Docker Desktop or install separately](https://docs.docker.com/compose/install/))

**Steps:**

1.  **Clone the repository:**
    ```bash
    git clone <repository-url>
    cd linkshrink
    ```

2.  **Create Environment File:**
    Copy the example environment file and edit it.
    ```bash
    cp .env.example .env
    nano .env # Or use your preferred editor
    ```
    *   **Crucial:** Set a secure `JWT_SECRET`.

3.  **Build and Run:**
    Use Docker Compose to build the application image and start the application and database containers.
    ```bash
    docker compose build
    docker compose up -d # -d runs the containers in detached mode (background)
    ```
    *   **Note:** This setup uses `network_mode: host`. Ensure ports `5432` and `8080` (or `APP_PORT`) are free on your host machine.
    *   The application should be accessible at `http://localhost:8080`.

4.  **View Logs (Optional):**
    ```bash
    docker compose logs -f app # View application logs
    docker compose logs -f db  # View database logs
    ```

5.  **Stopping the Services:**
    ```bash
    docker compose down
    ```
    To also remove the database volume (lose data): `docker compose down -v`

## API Usage

*   **Register:** `POST /register` (Body: `{"username": "user", "password": "pass"}`)
*   **Login:** `POST /login` (Body: `{"username": "user", "password": "pass"}`) -> Returns a JWT token.

*   **(Authenticated)** **Create Paid Route:** `POST /links/shrink`
    *   Requires `Authorization: Bearer <token>` header.
    *   Body: `{"target_url": "https://api.example.com/data", "method": "POST", "price": "0.00001"}`
    *   Returns details of the created route including `short_code` and `access_url`.

*   **(Authenticated)** **List Paid Routes:** `GET /links`
    *   Requires `Authorization: Bearer <token>` header.
    *   Returns a list of routes created by the user.

*   **(Authenticated)** **Delete Paid Route:** `DELETE /links/:linkID`
    *   Requires `Authorization: Bearer <token>` header.
    *   Deletes the route with the specified ID (if owned by the user).

*   **Access Paid Route:** `ANY /<shortCode>` (e.g., `POST /aBc1De2`)
    *   Requires appropriate L402 payment header (`Authorization: L402 ...` or legacy `X-PAYMENT`) - **Verification logic is currently placeholder/incomplete.**
    *   If payment is valid (or verification skipped), proxies the request (method, headers, body) to the configured `target_url`.
    *   Returns the response from the target service.

## Development Notes

*   The core payment verification logic in `internal/api/handlers/paid_route_handler.go` (`HandlePaidRoute`) and the L402 token parsing/handling in `internal/x402/middleware.go` (`VerifyX402Payment`) need implementation.
*   The current implementation assumes the `price` stored for a route is the direct crypto amount (e.g., ETH). Real-world use might require conversion from other currencies (e.g., USD) using an oracle/price feed.
