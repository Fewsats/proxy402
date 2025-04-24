# LinkShrink URL Shortener

LinkShrink is a simple URL shortening service written in Go. It allows registered users to create short links that redirect to original URLs. The service uses PostgreSQL to store user and link data, and JWT for user authentication.

## Features

*   User registration and login
*   JWT-based authentication for protected endpoints
*   URL shortening (`/shrink` endpoint)
*   Redirection from short code to original URL (`/:shortCode` endpoint)
*   Link expiration (optional)
*   Visit count tracking
*   List user's links (`/links` endpoint)
*   Delete user's links (`/links/:linkID` endpoint)

## Tech Stack

*   **Language:** Go
*   **Web Framework:** Gin Gonic
*   **Database:** PostgreSQL
*   **ORM:** GORM
*   **Authentication:** JWT (golang-jwt/jwt)
*   **Password Hashing:** bcrypt
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
    Copy the example environment file and **edit it to set a secure `JWT_SECRET`**.
    ```bash
    cp .env.example .env
    nano .env # Or use your preferred editor
    ```
    *Important:* Change `JWT_SECRET` to a long, random, and secure string. You can also adjust database credentials or ports if needed, but the defaults work with the `docker-compose.yml` setup.

3.  **Build and Run:**
    Use Docker Compose to build the application image and start the application and database containers.
    ```bash
    docker compose build
    docker compose up -d # -d runs the containers in detached mode (background)
    ```
    The application should now be running and accessible, typically at `http://localhost:8080` (or the `APP_PORT` you specified in `.env`).

4.  **View Logs (Optional):**
    If you ran with `-d`, you can view the logs:
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

Once the service is running, you can interact with it using tools like `curl`, Postman, or Insomnia.

*   **Register:** `POST /register` (Body: `{"username": "user", "password": "pass"}`)
*   **Login:** `POST /login` (Body: `{"username": "user", "password": "pass"}`) -> Returns a JWT token.
*   **Shrink URL:** `POST /shrink` (Requires `Authorization: Bearer <token>` header. Body: `{"original_url": "https://example.com"}`)
*   **Redirect:** `GET /<shortCode>` (e.g., `GET /aBc1De2`)
*   **Get User Links:** `GET /links` (Requires `Authorization: Bearer <token>` header)
*   **Delete Link:** `DELETE /links/<linkID>` (Requires `Authorization: Bearer <token>` header)
